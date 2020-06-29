/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/go-resty/resty/v2"
	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	"gitlab.com/cogment/cogment/api"
	"gitlab.com/cogment/cogment/deployment"
	"gitlab.com/cogment/cogment/helper"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var colorize = []func(format string, a ...interface{}) string{
	color.BlackString,
	color.RedString,
	color.GreenString,
	color.YellowString,
	color.BlueString,
	color.MagentaString,
	color.CyanString,
	color.WhiteString,
}

var coloredService = make(map[string]func(format string, a ...interface{}) string)

type logsCmdContext struct {
	from string
	to   string
	tail int
}

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:    "logs [options] [SERVICE...]",
	Short:  "Fetch logs of an application",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := deployment.PlatformClient(Verbose)
		if err != nil {
			log.Fatal(err)
		}

		var output []string
		row := []string{"SERVICE", "TIMESTAMP", "MESSAGE"}
		output = append(output, strings.Join(row, "|"))
		out := columnize.SimpleFormat(output)
		fmt.Println(out)

		tail := 0
		if cmd.Flags().Changed("tail") {
			if tail, err = cmd.Flags().GetInt("tail"); err != nil {
				log.Fatal(err)

			}
		}

		to := ""
		if cmd.Flags().Changed("to") {
			if to, err = cmd.Flags().GetString("to"); err != nil {
				log.Fatal(err)
			}
		}

		lastFrom := ""
		if cmd.Flags().Changed("from") {
			if lastFrom, err = cmd.Flags().GetString("from"); err != nil {
				log.Fatal(err)
			}
		}

		var lastTs time.Time
		for {

			ctx := logsCmdContext{
				from: lastFrom,
				to:   to,
				tail: tail,
			}

			result, err := getLogs(client, args, &ctx)
			if err != nil {
				log.Fatal(err)
			}

			for _, entry := range result.Results {
				service := entry.ServiceName

				sec, dec := math.Modf(entry.Timestamp)
				dec = math.Ceil(dec*1000) / 1000
				ts := time.Unix(int64(sec), int64(dec*(1e9))).UTC()

				row := []string{
					getColoredService(service),
					ts.Format("2006-01-02T15:04:05.000Z"),
					entry.Message,
				}

				fmt.Println(columnize.SimpleFormat([]string{strings.Join(row, "|")}))
				lastTs = ts
			}

			if !cmd.Flags().Changed("follow") || cmd.Flags().Changed("to") {
				return
			}

			if !lastTs.IsZero() {
				shift, err := time.ParseDuration("1ms")
				if err != nil {
					log.Fatal(err)
				}
				lastFrom = lastTs.Add(shift).Format("2006-01-02T15:04:05.000Z")
			}

			tail = 0
			time.Sleep(2 * time.Second)
		}
	},
}

func getColoredService(service string) string {
	colorIdx := int(math.Mod(float64(len(coloredService)), float64(len(colorize))))

	if _, ok := coloredService[service]; !ok {
		coloredService[service] = colorize[colorIdx]
	}

	return coloredService[service](service)

}

func getLogs(client *resty.Client, args []string, ctx *logsCmdContext) (*api.SearchLogsResult, error) {
	appId := helper.CurrentConfig("app")
	if appId == "" {
		log.Fatal("No current application found, maybe try `cogment new`")
	}

	request := client.R().SetResult(&api.SearchLogsResult{})

	if ctx.from != "" {
		request = request.SetQueryParam("from_time", ctx.from)
	}

	if ctx.to != "" {
		request = request.SetQueryParam("to_time", ctx.to)
	}

	if ctx.tail > 0 {
		request = request.SetQueryParam("tail", strconv.Itoa(ctx.tail))
	}

	if len(args) > 0 {
		services := strings.Join(args, ",")
		request = request.SetQueryParam("services", services)
	}

	resp, err := request.Get("/applications/" + appId + "/logs")

	if err != nil {
		log.Fatalf("%v", err)
	}

	if http.StatusNotFound == resp.StatusCode() {
		return nil, fmt.Errorf("%s", "Application not found")
	}

	if http.StatusOK == resp.StatusCode() {
		result := resp.Result().(*api.SearchLogsResult)
		return result, nil
	}

	return nil, fmt.Errorf("%s", resp.Body())
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().Int("tail", 0, "Number of lines to show from the end of the logs, Value is expected to be a UTC datetime in RFC3339Nano format (example: 2019-10-22T17:47:44.786585665Z)")
	logsCmd.Flags().String("from", "", "Starting date of the logs, Value is expected to be a UTC datetime in RFC3339Nano format (example: 2019-10-22T17:47:44.786585665Z)")
	logsCmd.Flags().String("to", "", "Ending date of the logs")

}
