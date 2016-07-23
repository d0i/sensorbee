package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/sensorbee/sensorbee.v0/version"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

func main() {
	app := cli.NewApp()
	app.Name = "build_sensorbee"
	app.Usage = "Build a custom sensorbee command"
	app.Version = version.Version
	binaryName := "sensorbee"
	if runtime.GOOS == "windows" {
		binaryName = "sensorbee.exe"
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "build.yaml",
			Usage: "path to a config file",
		},
		cli.StringFlag{
			Name:  "out, o",
			Value: binaryName,
			Usage: "the filename of the custom sensorbee command",
		},
		cli.StringFlag{
			Name:  "source-filename",
			Value: "sensorbee_main.go",
			Usage: "the name of the filename containing func main() generated by build_sensorbee",
		},
		cli.BoolTFlag{
			Name:  "download-plugins",
			Usage: "download all plugins",
		},
		cli.BoolFlag{
			Name:  "only-generate-source",
			Usage: "only generating a main source file and not building a binary",
		},
		cli.BoolFlag{
			Name: "verbose",
			Usage: "redundunt messages",
		},
	}
	app.Action = func(c *cli.Context) error {
		if err := action(c); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}
	app.Run(os.Args)
}

func action(c *cli.Context) (retErr error) {
	defer func() {
		if e := recover(); e != nil {
			retErr = fmt.Errorf("build_sensorbee failed with panic: %v", e)
		}
	}()

	err := func() error {
		if fn := c.String("source-filename"); fn != filepath.Base(fn) {
			return fmt.Errorf("the output file name must only contain a filename: %v", fn)
		}
		config, err := loadConfig(c.String("config"))
		if err != nil {
			return err
		}
		fmt.Printf("config loaded.\n")
		if err := downloadPlugins(c, config); err != nil {
			return err
		}
		if err := create(c, config); err != nil {
			return err
		}
		fmt.Printf("project creation succeeded.\n")
		if err := build(c, config); err != nil {
			return err
		}
		fmt.Printf("build finished.\n")
		return nil
	}()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

type Config struct {
	PluginPaths []string `yaml:"plugins"`
	SubCommands []string `yaml:"-"`
	Version     string   `yaml:"-"`
}

func loadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot load the config file '%v': %v", path, err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(b, config); err != nil {
		return nil, fmt.Errorf("cannot parse the config file '%v': %v", path, err)
	}
	// TODO: validation

	config.SubCommands = []string{"run", "shell", "topology", "exp", "runfile"}
	// TODO: sub commands should be configurable
	config.Version = version.Version
	return config, nil
}

func downloadPlugins(c *cli.Context, config *Config) error {
	if !c.BoolT("download-plugins") {
		return nil
	}

	// update main SensorBee
	fmt.Printf("updating sensorbee packages...")
	cmd := exec.Command("go", "get", "-u", "gopkg.in/sensorbee/sensorbee.v0/...")
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		b, _ := ioutil.ReadAll(buf)
		return fmt.Errorf("cannot get SensorBee core files: %v \n\n%v", err, string(b))
	}
	// download plugins
	fmt.Printf("downloading plugins...")
	for _, p := range config.PluginPaths {
		cmd := exec.Command("go", "get", "-u", p)
		buf := bytes.NewBuffer(nil)
		cmd.Stdout = buf
		cmd.Stderr = buf
		if err := cmd.Run(); err != nil {
			b, _ := ioutil.ReadAll(buf)
			return fmt.Errorf("cannot get a plugin '%v': %v \n\n%v", p, err, string(b))
		}
		fmt.Printf("%s successfully downloaded.", p)
	}
	return nil
}

func create(c *cli.Context, config *Config) error {
	tpl := template.Must(template.New("tpl").Parse(mainGoTemplate))
	var b bytes.Buffer
	if err := tpl.Execute(&b, config); err != nil {
		return fmt.Errorf("cannot generate a template source code: %v", err)
	}

	srcFile := c.String("source-filename")
	if err := ioutil.WriteFile(srcFile, b.Bytes(), 0644); err != nil {
		return fmt.Errorf("cannot generate an output file '%v': %v", srcFile, err)
	}

	// go fmt
	cmd := exec.Command("go", "fmt", srcFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot apply go fmt to the generated file: %v", err)
	}
	return nil
}

func build(c *cli.Context, config *Config) error {
	if c.Bool("only-generate-source") {
		fmt.Println("The custom command isn't built yet. Run the command below to build it:")
		fmt.Printf("go build -o \"%v\" %v\n", c.String("out"), c.String("source-filename"))
		return nil
	}
	cmd := exec.Command("go", "build", "-o", c.String("out"), c.String("source-filename"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot build a custom sensorbee command: %v", err)
	}
	return nil
}

const (
	mainGoTemplate = `package main

import (
	"github.com/codegangsta/cli"
	"os"
	"gopkg.in/sensorbee/sensorbee.v0/version"
	_ "gopkg.in/sensorbee/sensorbee.v0/bql/udf/builtin"{{range $_, $sub := .SubCommands}}
	"gopkg.in/sensorbee/sensorbee.v0/cmd/lib/{{$sub}}"{{end}}
	"time"
{{range $_, $path := .PluginPaths}}	_ "{{$path}}"
{{end}})

func init() {
	// TODO
	time.Local = time.UTC
}

func main() {
	app := cli.NewApp()
	app.Name = "sensorbee"
	app.Usage = "SensorBee built with build_sensorbee {{.Version}}"
	app.Version = version.Version
	app.Commands = []cli.Command{
{{range $_, $sub := .SubCommands}}		{{$sub}}.SetUp(),
{{end}}}
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
`
)
