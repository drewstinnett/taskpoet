package cmd

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var localClient *taskpoet.LocalClient
var dbConfig *taskpoet.DBConfig
var taskDefaults *taskpoet.Task

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "taskpoet",
	Short: "Do task tracking similiar to the olden ways of TaskWarrior",
	Long: `Designed to be similar to TaskWarrior, with some updated features, and specifics
around the Tom Limoncelli methods to task management`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.taskpoet.yaml)")

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

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugln("Using config file:", viper.ConfigFileUsed())
	}
	// Configuration for DB
	dbPath := viper.GetString("dbpath")
	dbConfig = &taskpoet.DBConfig{Path: dbPath}
	log.Debug("Using DB at: ", dbPath)

	err = taskpoet.InitDB(dbConfig)
	CheckErr(err)

	localClient, err = taskpoet.NewLocalClient(dbConfig)
	CheckErr(err)

	// Declare defaults
	taskDefaults = &taskpoet.Task{}
	defaultDue := viper.GetString("defaults.due")
	if defaultDue != "" {
		now := time.Now()
		dueDuration, err := taskpoet.ParseDuration(defaultDue)
		CheckErr(err)
		due := now.Add(dueDuration)
		taskDefaults.Due = due
	}
}

func CheckErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
