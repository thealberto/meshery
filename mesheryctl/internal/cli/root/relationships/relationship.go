// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package relationships

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/layer5io/meshery/mesheryctl/internal/cli/root/config"
	"github.com/layer5io/meshery/mesheryctl/pkg/utils"
	"github.com/meshery/schemas/models/v1alpha3/relationship"
	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	availableSubcommands = []*cobra.Command{viewCmd, generateCmd, listCmd, searchCmd}
)

type MeshmodelRelationshipsAPIResponse struct {
	Page          int                                   `json:"page"`
	PageSize      int                                   `json:"page_size"`
	Count         int64                                 `json:"total_count"`
	Relationships []relationship.RelationshipDefinition `json:"relationships"`
}

type relationshipsData struct {
	Headers          []string
	Rows             [][]string
	Count            int64
	DisplayCountOnly bool
}

var RelationshipCmd = &cobra.Command{
	Use:   "relationship",
	Short: "Manage relationships",
	Long: `Generate, list, search and view relationship(s) and detailed information
Meshery uses relationships to define how interconnected components interact.
`,
	Example: `
// Display number of available relationships in Meshery
mesheryctl relationship --count

// Generate a relationship documentation 
mesheryctl exp relationship generate [flags]

// List available relationship(s)
mesheryctl exp relationship list [flags]

// Search for a specific relationship
mesheryctl exp relationship search [flags] [query-text]

// View a specific relationship
mesheryctl exp relationship view [model-name]
	`,
	Args: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetBool("count")
		if len(args) == 0 && !count {
			errMsg := "Usage: mesheryctl exp relationship [subcommand]\nRun 'mesheryctl exp relationship --help' to see detailed help message"
			return utils.ErrInvalidArgument(fmt.Errorf("no command specified. %s", errMsg))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		countFlag, _ := cmd.Flags().GetBool("count")
		if countFlag {
			mctlCfg, err := config.GetMesheryCtl(viper.GetViper())
			if err != nil {
				log.Fatalln(err, "error processing config")
			}

			baseUrl := mctlCfg.GetBaseMesheryURL()
			url := fmt.Sprintf("%s/api/meshmodels/relationships?page=1", baseUrl)
			models, err := fetchRelationships(url)

			if err != nil {
				return err
			}

			utils.DisplayCount("relationships", models.Count)

			return nil
		}

		if ok := utils.IsValidSubcommand(availableSubcommands, args[0]); !ok {
			return errors.New(utils.RelationshipsError(fmt.Sprintf("'%s' is an invalid subcommand. Please provide required options from [view]. Use 'mesheryctl exp relationship --help' to display usage guide.\n", args[0]), "relationship"))
		}
		_, err := config.GetMesheryCtl(viper.GetViper())
		if err != nil {
			return utils.ErrLoadConfig(err)
		}

		err = cmd.Usage()
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	RelationshipCmd.AddCommand(availableSubcommands...)
	RelationshipCmd.Flags().BoolP("count", "c", false, "(optional) Get the number of relationship(s) in total")
}

func fetchRelationships(url string) (*MeshmodelRelationshipsAPIResponse, error) {
	req, err := utils.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := utils.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	// defers the closing of the response body after its use, ensuring that the resources are properly released.
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	modelsResponse := &MeshmodelRelationshipsAPIResponse{}
	err = json.Unmarshal(data, modelsResponse)
	if err != nil {
		return nil, err
	}

	return modelsResponse, nil
}

func listRelationships(cmd *cobra.Command, data relationshipsData) error {

	if len(data.Rows) == 0 {
		// if no component is found
		fmt.Println("No relationship(s) found")
		return nil
	}

	utils.DisplayCount("models", data.Count)

	if data.DisplayCountOnly {
		return nil
	}

	if cmd.Flags().Changed("page") {
		utils.PrintToTable(data.Headers, data.Rows)
	} else {
		maxRowsPerPage := 25
		err := utils.HandlePagination(maxRowsPerPage, "relationships", data.Rows, data.Headers)
		if err != nil {
			utils.Log.Error(err)
			return err
		}
	}
	return nil
}
