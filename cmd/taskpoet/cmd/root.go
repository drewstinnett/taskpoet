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
	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/drewstinnett/taskpoet/v2/themes"
	"github.com/drewstinnett/taskpoet/v2/themes/solarized"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	dbPath    string
	namespace string
	poetC     *taskpoet.Poet
	verbose   bool
	version   string = "dev"
)

// defaultCmd is what runs when no command is given
const defaultCmd = "list"

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
4 - High Effort, Low Impact (Charity)

Coming from TaskWarrior? Bring everything with you:

$ taskpoet import --from-task

or, if you have an export file: taskpoet import tw.json`,
		Version: version,
		// Execute prints errors itself, and a not-found isn't a usage problem
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.taskpoet.yaml)")
	cmd.PersistentFlags().StringVar(&dbPath, "db", "", "path to the database file (default is $XDG_DATA_HOME/taskpoet/taskpoet.db)")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of tasks")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	panicIfErr(viper.BindPFlag("dbpath", cmd.PersistentFlags().Lookup("db")))
	addCmds(cmd,
		newAddCmd(),
		newAnnotateCmd(),
		newCompleteCmd(),
		newCompletedCmd(),
		newDeleteCmd(),
		newDescribeCmd(),
		newExportCmd(),
		newFakeitCmd(),
		newImportCmd(),
		newListCmd(),
		newLogCmd(),
		newRecurCmd(),
	)
	return cmd
}

func addCmds(cmd *cobra.Command, additionalCmds ...*cobra.Command) {
	for _, item := range additionalCmds {
		cmd.AddCommand(item)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := NewRootCmd()
	// default cmd if no cmd is given
	if cmd, _, err := rootCmd.Find(os.Args[1:]); err == nil && cmd == rootCmd && !wantsRootHelp(os.Args[1:]) {
		rootCmd.SetArgs(append([]string{defaultCmd}, os.Args[1:]...))
	}

	err := rootCmd.Execute()
	closePoet()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// wantsRootHelp is true when the user asked for the help or version of the
// program itself, rather than of the default command
func wantsRootHelp(args []string) bool {
	for _, a := range args {
		switch a {
		case "-h", "--help", "--version":
			return true
		}
	}
	return false
}

func init() {
	cobra.OnInitialize(initConfig)
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

		viper.AddConfigPath(home)
		viper.SetConfigName(".taskpoet")
	}

	viper.SetDefault("recurrence.enabled", true)
	viper.SetDefault("recurrence.limit", 1)
	viper.SetDefault("recurrence.catchup", string(taskpoet.CatchUpLatest))

	// TASKPOET_DBPATH, TASKPOET_THEME, TASKPOET_DEFAULTS_DUE and so on
	viper.SetEnvPrefix("taskpoet")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

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
}

// mustPoet opens the database the first time it is needed. Commands that
// don't touch tasks, like --help, never open it.
func mustPoet() *taskpoet.Poet {
	if poetC != nil {
		return poetC
	}
	opts := []taskpoet.Option{
		taskpoet.WithDatabasePath(viper.GetString("dbpath")),
		taskpoet.WithNamespace(namespace),
		taskpoet.WithStyling(getTheme(viper.GetString("theme"))),
	}
	if defaultDue := viper.GetString("defaults.due"); defaultDue != "" {
		dueDuration, err := taskpoet.ParseDuration(defaultDue)
		checkErr(err)
		opts = append(opts, taskpoet.WithDefaultDue(dueDuration))
	}
	catchUp, err := taskpoet.ParseCatchUp(viper.GetString("recurrence.catchup"))
	checkErr(err)
	opts = append(opts, taskpoet.WithRecurrence(viper.GetInt("recurrence.limit"), catchUp))
	p, err := taskpoet.New(opts...)
	checkErr(err)
	poetC = p
	return poetC
}

// spawnRecurring creates any recurring task instances that are due, unless
// the recurrence.enabled setting turned that off
func spawnRecurring() {
	if !viper.GetBool("recurrence.enabled") {
		return
	}
	rep, err := mustPoet().SpawnRecurring(false)
	checkErr(err)
	if len(rep.Created) > 0 {
		log.Debug("Created recurring tasks", "count", len(rep.Created))
	}
	for _, w := range rep.Warnings {
		log.Debug(w)
	}
}

func closePoet() {
	if poetC != nil {
		if err := poetC.Close(); err != nil {
			log.Error("closing database", "error", err)
		}
		poetC = nil
	}
}

func checkErr(err error) {
	if err != nil {
		closePoet()
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

// tableOptsWithCmd builds the table options from the flags and arguments
// shared by the commands that list tasks. Arguments are a case insensitive
// regex on the description.
func tableOptsWithCmd(cmd *cobra.Command, args []string) (*taskpoet.TableOpts, error) {
	opts := &taskpoet.TableOpts{
		FilterParams: taskpoet.FilterParams{
			Project: mustGetCmd[string](cmd, "project"),
			Tag:     mustGetCmd[string](cmd, "tag"),
			Limit:   mustGetCmd[int](cmd, "limit"),
		},
		Filters: []taskpoet.Filter{
			taskpoet.FilterRegex,
			taskpoet.FilterProject,
			taskpoet.FilterTag,
		},
	}
	if len(args) > 0 {
		re, err := regexp.Compile(fmt.Sprintf("(?i)%v", strings.Join(args, " ")))
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %w", err)
		}
		opts.FilterParams.Regex = re
		log.Debug("Showing tasks that match", "regex", re)
	}
	return opts, nil
}

func bindTableOpts(cmd *cobra.Command) {
	cmd.Flags().IntP("limit", "l", 40, "Limit to N results")
	cmd.Flags().StringP("project", "P", "", "Only show tasks in this project (and its subprojects)")
	cmd.Flags().StringP("tag", "t", "", "Only show tasks with this tag")
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
	return mustPoet().CompleteIDs(toComplete, taskpoet.StatusPending), cobra.ShellCompDirectiveNoFileComp
}

func completeAny(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return mustPoet().CompleteIDs(toComplete), cobra.ShellCompDirectiveNoFileComp
}
