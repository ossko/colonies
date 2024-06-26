package cli

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/colonyos/colonies/pkg/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	generatorCmd.AddCommand(addGeneratorCmd)
	generatorCmd.AddCommand(packGeneratorCmd)
	generatorCmd.AddCommand(delGeneratorCmd)
	generatorCmd.AddCommand(getGeneratorCmd)
	generatorCmd.AddCommand(getGeneratorsCmd)
	rootCmd.AddCommand(generatorCmd)

	generatorCmd.PersistentFlags().StringVarP(&ServerHost, "host", "", "localhost", "Server host")
	generatorCmd.PersistentFlags().IntVarP(&ServerPort, "port", "", -1, "Server HTTP port")

	addGeneratorCmd.Flags().StringVarP(&PrvKey, "prvkey", "", "", "Private key")
	addGeneratorCmd.Flags().StringVarP(&SpecFile, "spec", "", "", "JSON specification of a Colony workflow")
	addGeneratorCmd.MarkFlagRequired("spec")
	addGeneratorCmd.Flags().StringVarP(&ColonyName, "colonyid", "", "", "Colony Id")
	addGeneratorCmd.Flags().StringVarP(&GeneratorName, "name", "", "", "Generator name")
	addGeneratorCmd.MarkFlagRequired("name")
	addGeneratorCmd.Flags().IntVarP(&GeneratorTrigger, "trigger", "", -1, "Trigger")
	addGeneratorCmd.MarkFlagRequired("trigger")
	addGeneratorCmd.Flags().IntVarP(&GeneratorTimeout, "timeout", "", -1, "Timeout")

	packGeneratorCmd.Flags().StringVarP(&PrvKey, "prvkey", "", "", "Private key")
	packGeneratorCmd.Flags().StringVarP(&GeneratorID, "generatorid", "", "", "Generator Id")
	packGeneratorCmd.MarkFlagRequired("generatorid")
	packGeneratorCmd.Flags().StringVarP(&Arg, "arg", "", "", "Arg to pack to generator")
	packGeneratorCmd.MarkFlagRequired("arg")

	delGeneratorCmd.Flags().StringVarP(&PrvKey, "prvkey", "", "", "Private key")
	delGeneratorCmd.Flags().StringVarP(&GeneratorID, "generatorid", "", "", "Generator Id")
	delGeneratorCmd.MarkFlagRequired("generatorid")

	getGeneratorCmd.Flags().StringVarP(&PrvKey, "prvkey", "", "", "Private key")
	getGeneratorCmd.Flags().StringVarP(&GeneratorID, "generatorid", "", "", "Generator Id")
	getGeneratorCmd.MarkFlagRequired("generatorid")

	getGeneratorsCmd.Flags().StringVarP(&PrvKey, "prvkey", "", "", "Private key")
	getGeneratorsCmd.Flags().StringVarP(&ColonyName, "colonyid", "", "", "Colony Id")
	getGeneratorsCmd.Flags().IntVarP(&Count, "count", "", DefaultCount, "Number of generators to list")
}

var generatorCmd = &cobra.Command{
	Use:   "generator",
	Short: "Manage generators",
	Long:  "Manage generators",
}

var addGeneratorCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a generator",
	Long:  "Add a generator",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		jsonSpecBytes, err := ioutil.ReadFile(SpecFile)
		CheckError(err)

		jsonStr := "{\"functionspecs\":" + string(jsonSpecBytes) + "}"
		workflowSpec, err := core.ConvertJSONToWorkflowSpec(jsonStr)
		CheckError(err)

		if workflowSpec.ColonyName == "" {
			workflowSpec.ColonyName = ColonyName
		}

		workflowSpecJSON, err := workflowSpec.ToJSON()
		CheckError(err)

		if GeneratorName == "" {
			CheckError(errors.New("Generator name not specified"))
		}

		if GeneratorTrigger == -1 {
			CheckError(errors.New("Generator trigger not specified"))
		}

		generator := core.CreateGenerator(ColonyName, GeneratorName, workflowSpecJSON, GeneratorTrigger, GeneratorTimeout)
		addedGenerator, err := client.AddGenerator(generator, PrvKey)
		CheckError(err)

		log.WithFields(log.Fields{"GeneratorID": addedGenerator.ID, "GeneratorName": GeneratorName, "Trigger": GeneratorTrigger, "Timeout": GeneratorTimeout}).Info("Generator added")
	},
}

var packGeneratorCmd = &cobra.Command{
	Use:   "pack",
	Short: "Pack arg to a generator",
	Long:  "Pack arg to a generator",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		if GeneratorID == "" {
			CheckError(errors.New("Generator Id not specified"))
		}

		err := client.PackGenerator(GeneratorID, Arg, PrvKey)
		CheckError(err)

		log.WithFields(log.Fields{"GeneratorID": GeneratorID, "Arg": Arg}).Info("Packing arg to generator")
	},
}

var delGeneratorCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a generator",
	Long:  "Remove a generator",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		if GeneratorID == "" {
			CheckError(errors.New("Generator Id not specified"))
		}

		err := client.RemoveGenerator(GeneratorID, PrvKey)
		CheckError(err)

		log.WithFields(log.Fields{"GeneratorID": GeneratorID}).Info("Removing generator")
	},
}

var getGeneratorCmd = &cobra.Command{
	Use:   "get",
	Short: "Get info about a generator",
	Long:  "Get info about a generator",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		if GeneratorID == "" {
			CheckError(errors.New("Generator Id not specified"))
		}

		generator, err := client.GetGenerator(GeneratorID, PrvKey)
		if generator == nil {
			log.WithFields(log.Fields{"GeneratorId": GeneratorID}).Error("Generator not found")
			os.Exit(0)
		}

		printGeneratorTable(generator)

		workflowSpec, err := core.ConvertJSONToWorkflowSpec(generator.WorkflowSpec)
		CheckError(err)
		for _, funcSpec := range workflowSpec.FunctionSpecs {
			printFunctionSpecTable(&funcSpec)
		}
	},
}

var getGeneratorsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all generators",
	Long:  "List all generators",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		generators, err := client.GetGenerators(ColonyName, Count, PrvKey)
		CheckError(err)

		if len(generators) == 0 {
			log.Info("No generators found")
			os.Exit(0)
		}

		printGeneratorsTable(generators)
	},
}
