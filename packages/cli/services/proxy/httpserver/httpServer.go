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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cogment/cogment/services/proxy/actor"
	"github.com/cogment/cogment/services/proxy/controller"
	"github.com/cogment/cogment/services/proxy/trialspec"
	"github.com/cogment/cogment/version"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const ActorTrialTokenHeaderKey = "Cogment-Actor-Trial-Token"

type response struct {
	Message string `json:"message"`
}

type Server struct {
	http.Server
	actorManager *actor.Manager
	controller   *controller.Controller
	tokenSecret  string
}

func New(
	port uint,
	actorManager *actor.Manager,
	controller *controller.Controller,
	tokenSecret string,
) (*Server, error) {
	// Debug mode can be helpful during development
	gin.SetMode(gin.ReleaseMode)
	//gin.SetMode(gin.DebugMode)

	router := gin.New()

	server := &Server{
		Server: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
		actorManager: actorManager,
		controller:   controller,
		tokenSecret:  tokenSecret,
	}

	router.HandleMethodNotAllowed = true

	// Allows all origins
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders(ActorTrialTokenHeaderKey)

	router.Use(cors.New(corsConfig))

	// Use a custom error handler
	router.Use(ginErrorHandlerMiddleware)

	// Use the custom logger middleware
	router.Use(ginLoggerMiddleware)

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())

	router.GET("/", server.getInfo)
	router.GET("/controller/trials", server.listTrials)
	router.POST("/controller/trials", server.startTrial)
	router.POST("/actor/:actor_name/:trial_id", server.joinTrial)
	router.DELETE("/actor/:actor_name/:trial_id", server.leaveTrial)
	router.POST("/actor/:actor_name/:trial_id/:tick_id", server.act)
	router.GET("/actor/:actor_name/:trial_id/:tick_id", server.act)

	router.NoRoute(func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found"))
	})

	router.NoMethod(func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
	})

	return server, nil
}

type infoResponse struct {
	response
	Version     string `json:"version"`
	VersionHash string `json:"version_hash"`
}

func (server *Server) getInfo(c *gin.Context) {
	c.JSON(http.StatusOK, infoResponse{
		response: response{
			Message: "This is the Cogment Web Proxy",
		},
		Version:     version.Version,
		VersionHash: version.Hash,
	})
}

func (server *Server) listTrials(c *gin.Context) {
	trials, err := server.controller.GetActiveTrials()
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, trials)
}

type startTrialRequest struct {
	TrialID string `json:"trial_id"`
	*controller.TrialParams
}

type startTrialResponse struct {
	response
	TrialID string `json:"trial_id"`
}

func (server *Server) startTrial(c *gin.Context) {
	request := startTrialRequest{
		TrialParams: controller.NewTrialParams(server.controller.Spec()),
	}

	err := c.ShouldBindJSON(&request)
	if err != nil && err != io.EOF {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	log := log.WithFields(logrus.Fields{
		"nb_actors": len(request.Actors),
	})
	if request.TrialID != "" {
		log = log.WithField("trial_id", request.TrialID)
	}
	log.Info("starting trial")

	trialID, err := server.controller.StartTrial(c, request.TrialID, request.TrialParams.TrialParams)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, startTrialResponse{
		response: response{
			Message: fmt.Sprintf("Trial [%s] started", trialID),
		},
		TrialID: trialID,
	})
}

func (server *Server) retrieveAndCheckActorTrial(c *gin.Context) (string, string, bool) {
	serializedToken := c.GetHeader(ActorTrialTokenHeaderKey)
	if serializedToken == "" {
		_ = c.AbortWithError(
			http.StatusBadRequest,
			fmt.Errorf("Missing require header [%s]", ActorTrialTokenHeaderKey),
		)
		return "", "", false
	}

	claims, err := ParseAndVerifyToken(serializedToken, server.tokenSecret)
	if err != nil {
		_ = c.AbortWithError(
			http.StatusUnauthorized,
			fmt.Errorf("Unable to validate token from header [%s] (%w)", ActorTrialTokenHeaderKey, err),
		)
		return "", "", false
	}

	trialID := c.Param("trial_id")
	actorName := c.Param("actor_name")

	if claims.TrialID != trialID || claims.ActorName != actorName {
		_ = c.AbortWithError(
			http.StatusUnauthorized,
			fmt.Errorf("Provided token doesn't match actor [%s] in trial [%s]", actorName, trialID),
		)
		return "", "", false
	}
	return trialID, actorName, true
}

type joinTrialResponse struct {
	response
	actor.JoinTrialResult
	Token string `json:"token"`
}

func (server *Server) joinTrial(c *gin.Context) {
	trialID := c.Param("trial_id")
	actorName := c.Param("actor_name")

	log.WithFields(logrus.Fields{
		"trial_id":   trialID,
		"actor_name": actorName,
	}).Info("joining trial")

	tokenStr, err := MakeAndSerializeToken(trialID, actorName, server.tokenSecret)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	res, err := server.actorManager.JoinTrial(c, trialID, actorName)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, joinTrialResponse{
		response: response{
			Message: fmt.Sprintf("Actor [%s] joined trial [%s]", actorName, trialID),
		},
		JoinTrialResult: res,
		Token:           tokenStr,
	})
}

func (server *Server) leaveTrial(c *gin.Context) {
	trialID, actorName, ok := server.retrieveAndCheckActorTrial(c)
	if !ok {
		return
	}
	log.WithFields(logrus.Fields{
		"trial_id":   trialID,
		"actor_name": actorName,
	}).Info("leaving trial")

	err := server.actorManager.LeaveTrial(trialID, actorName)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Message: fmt.Sprintf("Actor [%s] left trial [%s]", actorName, trialID),
	})
}

type actResponse struct {
	response
	actor.RecvEvent
	Action  *trialspec.DynamicPbMessage `json:"action"`
	Rewards []actor.SentReward          `json:"rewards,omitempty"`
}

func (server *Server) act(c *gin.Context) {
	trialID, actorName, ok := server.retrieveAndCheckActorTrial(c)
	if !ok {
		return
	}
	tickID, err := strconv.ParseUint(c.Param("tick_id"), 10, 64)
	if err != nil {
		_ = c.AbortWithError(
			http.StatusBadRequest,
			fmt.Errorf("%q is an invalid tick id, expecting a unsigned integer", c.Param("tick_id")),
		)
		return
	}

	actorTrial, err := server.actorManager.GetTrial(trialID, actorName)
	if err != nil {
		_ = c.AbortWithError(http.StatusNotFound, err)
		return
	}

	sentEvent := actor.SentEvent{}
	sentEvent.Action, err = server.actorManager.Spec().NewAction(actorTrial.ActorClass)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = c.ShouldBindJSON(&sentEvent)
	if err != nil && err != io.EOF {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	sentEvent.TickID = tickID

	log.WithFields(logrus.Fields{
		"trial_id":   trialID,
		"actor_name": actorName,
		"tick_id":    tickID,
	}).Info("acting")
	recvEvent, err := server.actorManager.Act(c, trialID, actorName, sentEvent)
	if err != nil {
		var trialEndedErr *actor.TrialEndedError
		if errors.As(err, &trialEndedErr) {
			c.JSON(http.StatusOK, response{
				Message: fmt.Sprintf("Trial [%s] ended", trialEndedErr.TrialID),
			})
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, actResponse{
		response: response{
			Message: fmt.Sprintf(
				"Actor [%s] submitted action in trial [%s] for tick [%d]",
				actorName,
				trialID,
				tickID,
			),
		},
		RecvEvent: recvEvent,
		Action:    sentEvent.Action,
		Rewards:   sentEvent.Rewards,
	})
}
