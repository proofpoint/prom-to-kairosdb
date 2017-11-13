package config

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
)

const defaultServerPort = ":9201"

// Config struct is top level config object
type Config struct {
	KairosdbURL          URL              `json:"kairosdb-url" yaml:"kairosdb-url"`
	MetricnamePrefix     string			  `json:"metricname-prefix" yaml:"metricname-prefix"`
	Timeout              time.Duration    `json:"timeout" yaml:"timeout"`
	MetricRelabelConfigs []*RelabelConfig `yaml:"metric_relabel_configs,omitempty"`
	Server               Server           `yaml:"server,omitempty"`
}

type Server struct {
	Port string `yaml:"port,flow,omitempty"`
}

// RelabelConfig defines the metric relabeling
type RelabelConfig struct {
	SourceLabels model.LabelNames `yaml:"source_labels,flow,omitempty"`
	Separator    string           `yaml:"separator,omitempty"`
	Regex        Regexp           `yaml:"regex,omitempty"`
	Action       RelabelAction    `yaml:"action,omitempty"`
	Prefix       string
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *RelabelConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c = &RelabelConfig{}
	type plain RelabelConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if c.Regex.Regexp == nil {
		c.Regex = MustNewRegexp("")
	}

	if c.Action == RelabelLabelDrop || c.Action == RelabelLabelKeep {
		if c.SourceLabels != nil ||
			c.Separator != "" ||
			c.Prefix != "" {
			return fmt.Errorf("%s action requires only 'regex', and no other fields", c.Action)
		}
	}

	if c.Action == RelabelAddPrefix {
		if c.Prefix == "" {
			return fmt.Errorf("%s action requires prefix", c.Action)
		}
	}

	return nil
}

// URL struct helps parse url from config file
type URL struct {
	*url.URL
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for URLs.
func (u *URL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	if err := unmarshal(&s); err != nil {
		return err
	}

	urlp, err := url.Parse(s)
	if err != nil {
		return err
	}
	u.URL = urlp
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface for URLs.
func (u URL) MarshalYAML() (interface{}, error) {
	if u.URL != nil {
		return u.String(), nil
	}
	return nil, nil
}

// Regexp is to contain the regular expression
type Regexp struct {
	*regexp.Regexp
	original string
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (re *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	r, err := NewRegexp(s)
	if err != nil {
		return err
	}
	*re = r
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (re Regexp) MarshalYAML() (interface{}, error) {
	if re.original != "" {
		return re.original, nil
	}
	return nil, nil
}

// NewRegexp creates a new anchored Regexp and returns an error if the
// passed-in regular expression does not compile.
func NewRegexp(s string) (Regexp, error) {
	regex, err := regexp.Compile("^(?:" + s + ")$")
	return Regexp{
		Regexp:   regex,
		original: s,
	}, err
}

// MustNewRegexp works like NewRegexp, but panics if the regular expression does not compile.
func MustNewRegexp(s string) Regexp {
	re, err := NewRegexp(s)
	if err != nil {
		panic(err)
	}
	return re
}

// RelabelAction is the action to be performed on relabeling.
type RelabelAction string

const (
	// RelabelKeep drops targets for which the input does not match the regex.
	RelabelKeep RelabelAction = "keep"
	// RelabelDrop drops targets for which the input does match the regex.
	RelabelDrop RelabelAction = "drop"
	// RelabelLabelDrop drops any label matching the regex.
	RelabelLabelDrop RelabelAction = "labeldrop"
	// RelabelLabelKeep drops any label not matching the regex.
	RelabelLabelKeep RelabelAction = "labelkeep"
	// RelabelAddPrefix adds prefix to the given labels
	RelabelAddPrefix RelabelAction = "addprefix"
)

// ParseCfgFile read the provided config file and parse it into config object
func ParseCfgFile(cfgFile string) (*Config, error) {
	if cfgFile == "" {
		return nil, fmt.Errorf("no config file provided")
	}

	filename, err := getAbsFilename(cfgFile)

	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, cfg)
	if err != nil {
		return nil, err
	}

	emptyurl := URL{}
	if cfg.KairosdbURL == emptyurl {
		return nil, fmt.Errorf("kairosdb-url is mandatory")
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = defaultServerPort
	}

	if cfg.MetricnamePrefix != "" {
		relabelConfig := &RelabelConfig{
			SourceLabels: model.LabelNames{model.MetricNameLabel},
			Regex: MustNewRegexp(".*"),
			Action: RelabelAddPrefix,
			Prefix: cfg.MetricnamePrefix,
		}

		cfg.MetricRelabelConfigs = append(cfg.MetricRelabelConfigs, relabelConfig)
	}

	return cfg, nil
}

func getAbsFilename(cfgFile string) (string, error) {
	cwd := getCurrentWorkingDirectory()
	home, _ := homedir.Dir()

	files := []string{
		cfgFile,
		fmt.Sprintf("%s/%s", cwd, cfgFile),
		fmt.Sprintf("%s/%s", home, cfgFile),
	}

	for _, file := range files {
		if ok, _ := ValidateFile(file); ok {
			return file, nil
		}
	}

	return "", fmt.Errorf("valid file not found")
}

func getCurrentWorkingDirectory() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}

func ValidateFile(cfgFile string) (bool, error) {
	fstat, err := os.Stat(cfgFile)
	if os.IsNotExist(err) {
		fmt.Printf("%s dont exist\n", cfgFile)
		return false, fmt.Errorf("file not found\n")
	}

	if fstat.IsDir() {
		fmt.Printf("%s is a directory, not a file\n", cfgFile)
		return false, fmt.Errorf("config file a directory, valid yaml file needed")
	}

	if fstat.Size() == 0 {
		fmt.Printf("%s is empty", cfgFile)
		return false, fmt.Errorf("config file is empty\n")
	}

	return true, nil
}
