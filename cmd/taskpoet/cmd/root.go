/*
Package cmd is the command line utility
*/
package cmd

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/drewstinnett/taskpoet/themes"
	"github.com/drewstinnett/taskpoet/themes/solarized"
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
	version   string = "dev"
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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
		Version:          version,
	}
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.taskpoet.yaml)")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of tasks")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	addCmds(cmd,
		newAddCmd(),
		newFakeitCmd(),
		newCommentCmd(),
		newCompleteCmd(),
		newCompletedCmd(),
		newDebugCmd(),
		newDescribeCmd(),
		newGetCmd(),
		newImportCmd(),
		newLogCmd(),
		newPluginsCmd(),
		newServerCmd(),
		newUICmd(),
	)
	return cmd
}

func addCmds(cmd *cobra.Command, additionalCmds ...*cobra.Command) {
	for _, item := range additionalCmds {
		cmd.AddCommand(item)
	}
}

// var rootCmd = NewRootCmd()

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// rootCmd := NewRootCmd()

	rootCmd := NewRootCmd()
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
	level := log.InfoLevel
	if verbose {
		level = log.DebugLevel
	}
	log.SetLevel(level)

	// If a config file is found, read it in.
	if cerr := viper.ReadInConfig(); cerr == nil {
		log.Debug("Using config file", "file", viper.ConfigFileUsed())
	}
	poetC, err = taskpoet.New(
		taskpoet.WithDatabasePath(viper.GetString("dbpath")),
		taskpoet.WithNamespace(namespace),
		taskpoet.WithStyling(getTheme(viper.GetString("theme"))),
	)
	checkErr(err)

	// Declare defaults
	// poetC.Default = taskpoet.Task{}
	defaultDue := viper.GetString("defaults.due")
	if defaultDue != "" {
		dueDuration, err := taskpoet.ParseDuration(defaultDue)
		checkErr(err)
		due := time.Now().Add(dueDuration)
		poetC.Default.Due = &due
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func noComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

// mustGetCmd uses generics to get a given flag with the appropriate Type from a cobra.Command
func mustGetCmd[T []int | []string | int | uint | string | bool | time.Duration](cmd *cobra.Command, s string) T {
	switch any(new(T)).(type) {
	case *int:
		item, err := cmd.Flags().GetInt(s)
		panicIfErr(err)
		return any(item).(T)
	case *uint:
		item, err := cmd.Flags().GetUint(s)
		panicIfErr(err)
		return any(item).(T)
	case *string:
		item, err := cmd.Flags().GetString(s)
		panicIfErr(err)
		return any(item).(T)
	case *bool:
		item, err := cmd.Flags().GetBool(s)
		panicIfErr(err)
		return any(item).(T)
	case *[]int:
		item, err := cmd.Flags().GetIntSlice(s)
		panicIfErr(err)
		return any(item).(T)
	case *[]string:
		item, err := cmd.Flags().GetStringSlice(s)
		panicIfErr(err)
		return any(item).(T)
	case *time.Duration:
		item, err := cmd.Flags().GetDuration(s)
		panicIfErr(err)
		return any(item).(T)
	default:
		panic(fmt.Sprintf("unexpected use of mustGetCmd: %v", reflect.TypeOf(s)))
	}
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func applyCobra(cmd *cobra.Command, args []string, opts *taskpoet.TableOpts) error {
	var err error
	if opts.FilterParams.Limit, err = cmd.PersistentFlags().GetInt("limit"); err != nil {
		return err
	}
	var re *regexp.Regexp
	if len(args) > 0 {
		opts.FilterParams.Regex = regexp.MustCompile(fmt.Sprintf("(?i)%v", strings.Join(args, " ")))
		log.Debug("Showing tasks that match", "regex", re)
	} else {
		opts.FilterParams.Regex = regexp.MustCompile(".*")
	}
	return nil
}

func mustTableOptsWithCmd(cmd *cobra.Command, args []string) *taskpoet.TableOpts {
	got, err := tableOptsWithCmd(cmd, args)
	if err != nil {
		panic(err)
	}
	return got
}

func tableOptsWithCmd(cmd *cobra.Command, args []string) (*taskpoet.TableOpts, error) {
	opts := &taskpoet.TableOpts{
		FilterParams: taskpoet.FilterParams{},
	}
	var err error
	if opts.FilterParams.Limit, err = cmd.PersistentFlags().GetInt("limit"); err != nil {
		return nil, err
	}
	var re *regexp.Regexp
	if len(args) > 0 {
		opts.FilterParams.Regex = regexp.MustCompile(fmt.Sprintf("(?i)%v", strings.Join(args, " ")))
		log.Debug("Showing tasks that match", "regex", re)
	} else {
		opts.FilterParams.Regex = regexp.MustCompile(".*")
	}
	return opts, nil
}

func bindTableOpts(cmd *cobra.Command) {
	cmd.PersistentFlags().IntP("limit", "l", 40, "Limit to N results")
}

// themeMap maps a string to Theme generators
var themeMap map[string]func() themes.Styling = map[string]func() themes.Styling{
	"default":         themes.New,
	"solarized-light": solarized.NewLight,
	"solarized-dark":  solarized.NewDark,
}

func getTheme(n string) themes.Styling {
	if t, ok := themeMap[n]; ok {
		return t()
	}
	return themes.New()
}

func completeActive(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return poetC.CompleteIDsWithPrefix("/active", toComplete), cobra.ShellCompDirectiveNoFileComp
}
