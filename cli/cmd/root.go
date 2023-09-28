/*
Package cmd is the command line utility
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/lmittmann/tint"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	namespace string
	poetC     *taskpoet.Poet
	verbose   bool
	// dbConfig     *taskpoet.DBConfig
	taskDefaults *taskpoet.Task
	// logger       *slog.Logger
)

// rootCmd represents the base command when called without any subcommands
// var rootCmd *cobra.Command

// NewRootCmd is the root command generator
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "taskpoet",
		Short: "Do task tracking similar to the olden ways of TaskWarrior",
		Long: `Designed to be similar to TaskWarrior, with some updated features, and specifics
around the Tom Limoncelli methods to task management.

Key Concepts:

Effort/Impact Assessment, based on Limoncelli concept

0 - Unset
1 - Low Effort, High Impact (Sweet Spot)
2 - High Effort, High Impact (Homework)
3 - Low Effort, Low Impact (Busywork)
4 - High Effort, Low Impact (Charity)`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
		},
	}
	return cmd
}

var rootCmd = NewRootCmd()

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// rootCmd := NewRootCmd()

	cmd, _, err := rootCmd.Find(os.Args[1:])
	// default cmd if no cmd is given
	if err == nil && cmd.Use == rootCmd.Use && cmd.Flags().Parse(os.Args[1:]) != pflag.ErrHelp {
		args := append([]string{"active"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// cobra.CheckErr(rootCmd.Execute())
}

// var rootCmd *cobra.Command
func init() {
	// rootCmd = NewRootCmd()
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.taskpoet.yaml)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of tasks")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".taskpoet")
	}

	viper.AutomaticEnv() // read in environment variables that match
	var err error

	// set global logger with custom options
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
		}),
	))

	// If a config file is found, read it in.
	if cerr := viper.ReadInConfig(); cerr == nil {
		slog.Debug("Using config file", "file", viper.ConfigFileUsed())
	}
	poetC, err = taskpoet.New(taskpoet.WithDatabasePath(viper.GetString("dbpath")), taskpoet.WithNamespace(namespace))
	checkErr(err)

	// Declare defaults
	taskDefaults = &taskpoet.Task{}
	defaultDue := viper.GetString("defaults.due")
	if defaultDue != "" {
		now := time.Now()
		dueDuration, err := taskpoet.ParseDuration(defaultDue)
		checkErr(err)
		due := now.Add(dueDuration)
		taskDefaults.Due = &due
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func noComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}
