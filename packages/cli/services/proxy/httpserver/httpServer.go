// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cogment/cogment/services/proxy/actor"
	"github.com/cogment/cogment/services/proxy/controller"
	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/version"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/sirupsen/logrus"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

var infos = openapi.Info{
	Title: "Cogment Web Proxy",
	Description: "The Cogment Web Proxy is designed to facilitate the use of Cogment with web-based components." +
		" It implements a JSON HTTP API that can be easily used from a web application.\n" +
		"\n" +
		"The API is composed of two groups of routes:\n" +
		"- [Actor](#tag/Actor)\n" +
		"- [Trial Lifecycle Control](#tag/Trial-Lifecycle-Control)\n",
	Version: version.Version,
}

type Server struct {
	http.Server
	actorManager *actor.Manager
	controller   *controller.Controller
	tokenSecret  string

	gin  *gin.Engine
	fizz *fizz.Fizz
}

//nolint:lll
func New(
	port uint,
	actorManager *actor.Manager,
	controller *controller.Controller,
	tokenSecret string,
) (*Server, error) {
	// Debug mode can be helpful during development
	gin.SetMode(gin.ReleaseMode)
	//gin.SetMode(gin.DebugMode)

	tonic.SetErrorHook(tonicErrorHook)

	ginEngine := gin.New()
	fizzEngine := fizz.NewFromEngine(ginEngine)

	server := &Server{
		Server: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: fizzEngine,
		},
		actorManager: actorManager,
		controller:   controller,
		tokenSecret:  tokenSecret,
		gin:          ginEngine,
		fizz:         fizzEngine,
	}

	server.gin.HandleMethodNotAllowed = true

	err := overrideTypes(server.fizz.Generator())
	if err != nil {
		return nil, err
	}

	// Allows all origins
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders(actorTrialTokenHeaderKey)

	server.fizz.Use(cors.New(corsConfig))

	// Use a custom error handler
	server.fizz.Use(ginErrorHandlerMiddleware)

	// Use the custom logger middleware
	server.fizz.Use(ginLoggerMiddleware)

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	server.fizz.Use(gin.Recovery())

	server.fizz.GET("/", []fizz.OperationOption{
		fizz.Summary("Retrieve information about this API"),
	}, tonic.Handler(server.getInfo, http.StatusOK))

	server.fizz.GET("/openapi.json", []fizz.OperationOption{
		fizz.Summary("Retrieve the open api specification"),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, server.fizz.OpenAPI(&infos, "json"))

	controllerGroup := server.fizz.Group(
		"/controller",
		"Trial Lifecycle Control",
		"Interact with the Cogment Orchestrator to control the lifetime of trials.",
	)
	controllerGroup.GET("/trials", []fizz.OperationOption{
		fizz.Summary("Retrieve currently active trials"),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, tonic.Handler(server.listTrials, http.StatusOK))

	controllerGroup.POST("/trials", []fizz.OperationOption{
		fizz.Summary("Start a trial"),
		fizz.Description("Start a trial from given trial parameters.\n" +
			"For further documentation of trial parameters, take a look at the dedicated [page](/docs/reference/parameters)\n" +
			"\n" +
			"> It is forbidden to pass [gRPC endpoint](/docs/reference/parameters#grpc-scheme) as part of the trial parameters here"),
		fizz.Response("400", "Bad server configuration or state", httpError{}, nil, nil),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, tonic.Handler(server.startTrial, http.StatusAccepted))

	actorGroup := server.fizz.Group(
		"/actor",
		"Actor",
		"Interact with a running trial as an actor",
	)
	actorGroup.POST("/:actor_name/:trial_id", []fizz.OperationOption{
		fizz.Summary("Join a trial"),
		fizz.Description("Join trial `trial_id` as actor `actor_name`, retrieve the initial observation and the actor trial token"),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, tonic.Handler(server.joinTrial, http.StatusOK))
	actorGroup.DELETE("/:actor_name/:trial_id", []fizz.OperationOption{
		fizz.Summary("Leave a trial"),
		fizz.Description("This explicitly closes the trial connection between the actor and the trial"),
		fizz.Response("401", "Invalid Trial Actor Token", httpError{}, nil, nil),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, tonic.Handler(server.leaveTrial, http.StatusAccepted))
	actorGroup.POST("/:actor_name/:trial_id/:tick_id", []fizz.OperationOption{
		fizz.Summary("Act in a trial"),
		fizz.Response("401", "Invalid Trial Actor Token", httpError{}, nil, nil),
		fizz.Response("404", "Trial not found or Actor not found in Trial", httpError{}, nil, nil),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, tonic.Handler(server.act, http.StatusOK))
	actorGroup.GET("/:actor_name/:trial_id/:tick_id", []fizz.OperationOption{
		fizz.Summary("Act, with an empty action, in a trial"),
		fizz.Description("Have the actor `actor_name` in trial `trial_id` act, with an empty action, at tick `tick_id`.\n" +
			"\n" +
			"This will send an action initialized with its [default protobuf value](https://protobuf.dev/programming-guides/proto3/#default)."),
		fizz.Response("401", "Invalid Trial Actor Token", httpError{}, nil, nil),
		fizz.Response("404", "Trial not found or Actor not found in Trial", httpError{}, nil, nil),
		fizz.Response("500", "Bad server configuration or state", httpError{}, nil, nil),
	}, tonic.Handler(server.actEmpty, http.StatusOK))

	ginEngine.NoRoute(func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found"))
	})

	ginEngine.NoMethod(func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
	})

	return server, nil
}

func (server *Server) GenerateOpenAPISpec(outputFile string) error {
	f, _ := os.Create(outputFile)
	defer f.Close()

	server.fizz.Generator().SetInfo(&infos)
	serializedJSON, err := json.MarshalIndent(server.fizz.Generator().API(), "", "\t")
	if err != nil {
		return err
	}
	_, err = f.Write(serializedJSON)
	if err != nil {
		return err
	}
	return nil
}

type response struct {
	Message string `json:"message" description:"Human-readable response description"`
}

type infoResponse struct {
	response
	Version     string `json:"version" description:"Cogment Version"`
	VersionHash string `json:"version_hash"`
}

func (server *Server) getInfo(*gin.Context) (infoResponse, error) {
	return infoResponse{
		response: response{
			Message: "This is the Cogment Web Proxy",
		},
		Version:     version.Version,
		VersionHash: version.Hash,
	}, nil
}

func (server *Server) listTrials(c *gin.Context) ([]controller.TrialInfo, error) {
	// TODO make the timeout configurable by the end user
	trials, err := server.controller.GetActiveTrials(c, 50*time.Millisecond)
	if err != nil {
		return nil, wrapError(http.StatusInternalServerError, err)
	}
	return trials, nil
}

//nolint:lll
type startTrialRequest struct {
	TrialID string `json:"trial_id" description:"The trial identifier requested for the new trial. It must be unique. If not empty, the Orchestrator will try to use this trial_id, otherwise, a UUID will be created."`
	controller.TrialParams
}

type startTrialResponse struct {
	response
	TrialID string `json:"trial_id" description:"The trial identifier"`
}

func (server *Server) startTrial(c *gin.Context, request *startTrialRequest) (*startTrialResponse, error) {
	log := log.WithFields(logrus.Fields{
		"nb_actors": len(request.Actors),
	})
	if request.TrialID != "" {
		log = log.WithField("trial_id", request.TrialID)
	}
	log.Info("starting trial")

	pbTrialParams, err := request.FullyUnmarshal(server.controller.Spec())
	if err != nil {
		return nil, wrapError(http.StatusBadRequest, err)
	}

	trialID, err := server.controller.StartTrial(c, request.TrialID, pbTrialParams)
	if err != nil {
		return nil, wrapError(http.StatusInternalServerError, err)
	}

	return &startTrialResponse{
		response: response{
			Message: fmt.Sprintf("Trial [%s] started", trialID),
		},
		TrialID: trialID,
	}, nil
}

type joinTrialRequest struct {
	TrialID   string `path:"trial_id" description:"The trial identifier"`
	ActorName string `path:"actor_name" description:"The actor name"`
}

//nolint:lll
type joinTrialResponse struct {
	response
	actor.JoinTrialResult
	Token string `json:"token" description:"Actor trial token, required to authenticate further interactions with the trial as this actor."`
}

func (server *Server) joinTrial(c *gin.Context, request *joinTrialRequest) (*joinTrialResponse, error) {
	trialID := request.TrialID
	actorName := request.ActorName

	log.WithFields(logrus.Fields{
		"trial_id":   trialID,
		"actor_name": actorName,
	}).Info("joining trial")

	tokenStr, err := MakeAndSerializeToken(trialID, actorName, server.tokenSecret)
	if err != nil {
		return nil, wrapError(http.StatusInternalServerError, err)
	}

	res, err := server.actorManager.JoinTrial(c, trialID, actorName)
	if err != nil {
		return nil, wrapError(http.StatusInternalServerError, err)
	}

	return &joinTrialResponse{
		response: response{
			Message: fmt.Sprintf("Actor [%s] joined trial [%s]", actorName, trialID),
		},
		JoinTrialResult: res,
		Token:           tokenStr,
	}, nil
}

const actorTrialTokenHeaderKey = "Cogment-Actor-Trial-Token"

//nolint:lll
type actorTrialRequest struct {
	TrialID   string `path:"trial_id" description:"The trial identifier"`
	ActorName string `path:"actor_name" description:"The actor name"`
	Token     string `header:"Cogment-Actor-Trial-Token" validate:"required" description:"The actor trial token, it must match the token returned when [joining the trial](#tag/Actor/operation/joinTrial-fm)."`
}

func (server *Server) checkActorTrialToken(request *actorTrialRequest) error {
	claims, err := ParseAndVerifyToken(request.Token, server.tokenSecret)
	if err != nil {
		return wrapError(
			http.StatusUnauthorized,
			fmt.Errorf("Unable to validate token from header [%s] (%w)", actorTrialTokenHeaderKey, err),
		)
	}

	if claims.TrialID != request.TrialID || claims.ActorName != request.ActorName {
		return wrapError(
			http.StatusUnauthorized,
			fmt.Errorf("Provided token doesn't match actor [%s] in trial [%s]", request.ActorName, request.TrialID),
		)
	}
	return nil
}

func (server *Server) leaveTrial(_ *gin.Context, request *actorTrialRequest) (*response, error) {
	err := server.checkActorTrialToken(request)
	if err != nil {
		return nil, err
	}

	log.WithFields(logrus.Fields{
		"trial_id":   request.TrialID,
		"actor_name": request.ActorName,
	}).Info("leaving trial")

	err = server.actorManager.LeaveTrial(request.TrialID, request.ActorName)
	if err != nil {
		// TODO: better handle errors
		return nil, wrapError(http.StatusInternalServerError, err)
	}

	return &response{
		Message: fmt.Sprintf("Actor [%s] left trial [%s]", request.ActorName, request.TrialID),
	}, nil
}

type actRequest struct {
	actorTrialRequest
	TickID  uint64             `path:"tick_id" description:"Identifier of the tick (time step)"`
	Action  *json.RawMessage   `json:"action" description:"Action from the actor"`
	Rewards []actor.SentReward `json:"rewards,omitempty" description:"Rewards sent by the actor"`
}

type actResponse struct {
	response
	actor.RecvEvent
	Action  *trialspec.DynamicPbMessage `json:"action" description:"Action from the actor"`
	Rewards []actor.SentReward          `json:"rewards,omitempty" description:"Rewards sent by the actor"`
}

func (server *Server) act(c *gin.Context, request *actRequest) (*actResponse, error) {
	err := server.checkActorTrialToken(&request.actorTrialRequest)
	if err != nil {
		return nil, err
	}

	actorTrial, err := server.actorManager.GetTrial(request.TrialID, request.ActorName)
	if err != nil {
		return nil, wrapError(http.StatusNotFound, err)
	}

	sentEvent := actor.SentEvent{
		TickID:  request.TickID,
		Rewards: request.Rewards,
	}
	sentEvent.Action, err = server.actorManager.Spec().NewAction(actorTrial.ActorClass)
	if err != nil {
		return nil, wrapError(http.StatusInternalServerError, err)
	}

	if err := json.Unmarshal(*request.Action, sentEvent.Action); err != nil {
		return nil, wrapError(http.StatusBadRequest, err)
	}

	return server.doAct(c, request.TrialID, request.ActorName, sentEvent)
}

func (server *Server) doAct(
	c *gin.Context,
	trialID string,
	actorName string,
	sentEvent actor.SentEvent,
) (*actResponse, error) {
	log.WithFields(logrus.Fields{
		"trial_id":   trialID,
		"actor_name": actorName,
		"tick_id":    sentEvent.TickID,
	}).Info("acting")
	recvEvent, err := server.actorManager.Act(c, trialID, actorName, sentEvent)
	if err != nil {
		var trialEndedErr *actor.TrialEndedError
		if errors.As(err, &trialEndedErr) {
			return &actResponse{
				response: response{
					Message: fmt.Sprintf("Trial [%s] ended", trialEndedErr.TrialID),
				},
			}, nil
		}
		// TODO better handle the other errors
		return nil, wrapError(http.StatusInternalServerError, err)
	}
	return &actResponse{
		response: response{
			Message: fmt.Sprintf(
				"Actor [%s] submitted action in trial [%s] for tick [%d]",
				actorName,
				trialID,
				sentEvent.TickID,
			),
		},
		RecvEvent: recvEvent,
		Action:    sentEvent.Action,
		Rewards:   sentEvent.Rewards,
	}, nil
}

type actEmptyRequest struct {
	actorTrialRequest
	TickID uint64 `path:"tick_id" description:"Identifier of the tick (time step)"`
}

func (server *Server) actEmpty(c *gin.Context, request *actEmptyRequest) (*actResponse, error) {
	err := server.checkActorTrialToken(&request.actorTrialRequest)
	if err != nil {
		return nil, err
	}

	actorTrial, err := server.actorManager.GetTrial(request.TrialID, request.ActorName)
	if err != nil {
		return nil, wrapError(http.StatusNotFound, err)
	}

	sentEvent := actor.SentEvent{
		TickID: request.TickID,
	}
	sentEvent.Action, err = server.actorManager.Spec().NewAction(actorTrial.ActorClass)
	if err != nil {
		return nil, wrapError(http.StatusInternalServerError, err)
	}

	return server.doAct(c, request.TrialID, request.ActorName, sentEvent)
}
