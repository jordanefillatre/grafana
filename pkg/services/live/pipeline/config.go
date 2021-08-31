package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/grafana/grafana/pkg/services/live/managedstream"

	"github.com/centrifugal/centrifuge"
)

// FileStorage can load channel rules from a file on disk.
type FileStorage struct {
	Node                *centrifuge.Node
	ManagedStream       *managedstream.Runner
	FrameStorage        *FrameStorage
	remoteWriteBackends []RemoteWriteBackend
}

type JsonAutoSettings struct{}

type ConverterConfig struct {
	Type                      string                     `json:"type"`
	AutoJsonConverterConfig   *AutoJsonConverterConfig   `json:"jsonAuto,omitempty"`
	ExactJsonConverterConfig  *ExactJsonConverterConfig  `json:"jsonExact,omitempty"`
	AutoInfluxConverterConfig *AutoInfluxConverterConfig `json:"influxAuto,omitempty"`
	JsonFrameConverterConfig  *JsonFrameConverterConfig  `json:"jsonFrame,omitempty"`
}

type ProcessorConfig struct {
	Type                      string                     `json:"type"`
	DropFieldsProcessorConfig *DropFieldsProcessorConfig `json:"dropFields,omitempty"`
	KeepFieldsProcessorConfig *KeepFieldsProcessorConfig `json:"keepFields,omitempty"`
	MultipleProcessorConfig   *MultipleProcessorConfig   `json:"multiple,omitempty"`
}

type MultipleProcessorConfig struct {
	Processors []ProcessorConfig `json:"processors"`
}

type MultipleOutputterConfig struct {
	Outputters []OutputterConfig `json:"outputters"`
}

type ManagedStreamOutputConfig struct{}

type ConditionalOutputConfig struct {
	Condition *ConditionCheckerConfig `json:"condition"`
	Outputter *OutputterConfig        `json:"outputter"`
}

type RemoteWriteOutputConfig struct {
	UID string `json:"uid"`
}

type OutputterConfig struct {
	Type                    string                     `json:"type"`
	ManagedStreamConfig     *ManagedStreamOutputConfig `json:"ManagedStream,omitempty"`
	MultipleOutputterConfig *MultipleOutputterConfig   `json:"multiple,omitempty"`
	RedirectOutputConfig    *RedirectOutputConfig      `json:"redirect,omitempty"`
	ConditionalOutputConfig *ConditionalOutputConfig   `json:"conditional,omitempty"`
	ThresholdOutputConfig   *ThresholdOutputConfig     `json:"threshold,omitempty"`
	RemoteWriteOutputConfig *RemoteWriteOutputConfig   `json:"remoteWrite,omitempty"`
	ChangeLogOutputConfig   *ChangeLogOutputConfig     `json:"changeLog,omitempty"`
}

type ChannelRuleSettings struct {
	Converter *ConverterConfig `json:"converter,omitempty"`
	Processor *ProcessorConfig `json:"processor,omitempty"`
	Outputter *OutputterConfig `json:"outputter,omitempty"`
}

type ChannelRule struct {
	Pattern  string              `json:"pattern"`
	Settings ChannelRuleSettings `json:"settings"`
}

type RemoteWriteBackend struct {
	UID      string             `json:"uid"`
	Settings *RemoteWriteConfig `json:"settings"`
}

type ChannelRules struct {
	Rules               []ChannelRule        `json:"rules"`
	RemoteWriteBackends []RemoteWriteBackend `json:"remoteWriteBackends"`
}

func (f *FileStorage) extractConverter(config *ConverterConfig) (Converter, error) {
	if config == nil {
		return nil, nil
	}
	missingConfiguration := fmt.Errorf("missing configuration for %s", config.Type)
	switch config.Type {
	case "jsonAuto":
		if config.AutoJsonConverterConfig == nil {
			return nil, missingConfiguration
		}
		return NewAutoJsonConverter(*config.AutoJsonConverterConfig), nil
	case "jsonExact":
		if config.ExactJsonConverterConfig == nil {
			return nil, missingConfiguration
		}
		return NewExactJsonConverter(*config.ExactJsonConverterConfig), nil
	case "jsonFrame":
		if config.JsonFrameConverterConfig == nil {
			return nil, missingConfiguration
		}
		return NewJsonFrameConverter(*config.JsonFrameConverterConfig), nil
	case "influxAuto":
		if config.AutoInfluxConverterConfig == nil {
			return nil, missingConfiguration
		}
		return NewAutoInfluxConverter(*config.AutoInfluxConverterConfig), nil
	default:
		return nil, fmt.Errorf("unknown converter type: %s", config.Type)
	}
}

func (f *FileStorage) extractProcessor(config *ProcessorConfig) (Processor, error) {
	if config == nil {
		return nil, nil
	}
	missingConfiguration := fmt.Errorf("missing configuration for %s", config.Type)
	switch config.Type {
	case "dropFields":
		if config.DropFieldsProcessorConfig == nil {
			return nil, missingConfiguration
		}
		return NewDropFieldsProcessor(*config.DropFieldsProcessorConfig), nil
	case "keepFields":
		if config.KeepFieldsProcessorConfig == nil {
			return nil, missingConfiguration
		}
		return NewKeepFieldsProcessor(*config.KeepFieldsProcessorConfig), nil
	case "multiple":
		if config.MultipleProcessorConfig == nil {
			return nil, missingConfiguration
		}
		var processors []Processor
		for _, outConf := range config.MultipleProcessorConfig.Processors {
			out := outConf
			proc, err := f.extractProcessor(&out)
			if err != nil {
				return nil, err
			}
			processors = append(processors, proc)
		}
		return NewMultipleProcessor(processors...), nil
	default:
		return nil, fmt.Errorf("unknown processor type: %s", config.Type)
	}
}

type MultipleConditionCheckerConfig struct {
	Type       ConditionType            `json:"type"`
	Conditions []ConditionCheckerConfig `json:"conditions"`
}

type NumberCompareConditionConfig struct {
	FieldName string          `json:"fieldName"`
	Op        NumberCompareOp `json:"op"`
	Value     float64         `json:"value"`
}

type ConditionCheckerConfig struct {
	Type                           string                          `json:"type"`
	MultipleConditionCheckerConfig *MultipleConditionCheckerConfig `json:"multiple,omitempty"`
	NumberCompareConditionConfig   *NumberCompareConditionConfig   `json:"numberCompare,omitempty"`
}

func (f *FileStorage) extractConditionChecker(config *ConditionCheckerConfig) (ConditionChecker, error) {
	if config == nil {
		return nil, nil
	}
	missingConfiguration := fmt.Errorf("missing configuration for %s", config.Type)
	switch config.Type {
	case "numberCompare":
		if config.NumberCompareConditionConfig == nil {
			return nil, missingConfiguration
		}
		c := *config.NumberCompareConditionConfig
		return NewNumberCompareCondition(c.FieldName, c.Op, c.Value), nil
	case "multiple":
		var conditions []ConditionChecker
		if config.MultipleConditionCheckerConfig == nil {
			return nil, missingConfiguration
		}
		for _, outConf := range config.MultipleConditionCheckerConfig.Conditions {
			out := outConf
			cond, err := f.extractConditionChecker(&out)
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		}
		return NewMultipleConditionChecker(config.MultipleConditionCheckerConfig.Type, conditions...), nil
	default:
		return nil, fmt.Errorf("unknown condition type: %s", config.Type)
	}
}

func (f *FileStorage) extractOutputter(config *OutputterConfig) (Outputter, error) {
	if config == nil {
		return nil, nil
	}
	missingConfiguration := fmt.Errorf("missing configuration for %s", config.Type)
	switch config.Type {
	case "redirect":
		if config.RedirectOutputConfig == nil {
			return nil, missingConfiguration
		}
		return NewRedirectOutput(*config.RedirectOutputConfig), nil
	case "multiple":
		if config.MultipleOutputterConfig == nil {
			return nil, missingConfiguration
		}
		var outputters []Outputter
		for _, outConf := range config.MultipleOutputterConfig.Outputters {
			out := outConf
			outputter, err := f.extractOutputter(&out)
			if err != nil {
				return nil, err
			}
			outputters = append(outputters, outputter)
		}
		return NewMultipleOutputter(outputters...), nil
	case "ManagedStream":
		return NewManagedStreamOutput(f.ManagedStream), nil
	case "localSubscribers":
		return NewLocalSubscribersOutput(f.Node), nil
	case "conditional":
		if config.ConditionalOutputConfig == nil {
			return nil, missingConfiguration
		}
		condition, err := f.extractConditionChecker(config.ConditionalOutputConfig.Condition)
		if err != nil {
			return nil, err
		}
		outputter, err := f.extractOutputter(config.ConditionalOutputConfig.Outputter)
		if err != nil {
			return nil, err
		}
		return NewConditionalOutput(condition, outputter), nil
	case "threshold":
		if config.ThresholdOutputConfig == nil {
			return nil, missingConfiguration
		}
		return NewThresholdOutput(f.FrameStorage, *config.ThresholdOutputConfig), nil
	case "remoteWrite":
		if config.RemoteWriteOutputConfig == nil {
			return nil, missingConfiguration
		}
		remoteWriteConfig, ok := f.getRemoteWriteConfig(config.RemoteWriteOutputConfig.UID)
		if !ok {
			return nil, fmt.Errorf("unknown remote write backend uid: %s", config.RemoteWriteOutputConfig.UID)
		}
		return NewRemoteWriteOutput(*remoteWriteConfig), nil
	case "changeLog":
		if config.ChangeLogOutputConfig == nil {
			return nil, missingConfiguration
		}
		return NewChangeLogOutput(f.FrameStorage, *config.ChangeLogOutputConfig), nil
	default:
		return nil, fmt.Errorf("unknown output type: %s", config.Type)
	}
}

func (f *FileStorage) getRemoteWriteConfig(uid string) (*RemoteWriteConfig, bool) {
	for _, rwb := range f.remoteWriteBackends {
		if rwb.UID == uid {
			return rwb.Settings, true
		}
	}
	return nil, false
}

func (f *FileStorage) ListChannelRules(_ context.Context, _ ListLiveChannelRuleCommand) ([]*LiveChannelRule, error) {
	ruleBytes, _ := ioutil.ReadFile(os.Getenv("GF_LIVE_CHANNEL_RULES_FILE"))
	var channelRules ChannelRules
	err := json.Unmarshal(ruleBytes, &channelRules)
	if err != nil {
		return nil, err
	}

	f.remoteWriteBackends = channelRules.RemoteWriteBackends

	var rules []*LiveChannelRule

	for _, ruleConfig := range channelRules.Rules {
		rule := &LiveChannelRule{
			Pattern: ruleConfig.Pattern,
		}
		var err error
		rule.Converter, err = f.extractConverter(ruleConfig.Settings.Converter)
		if err != nil {
			return nil, err
		}
		rule.Processor, err = f.extractProcessor(ruleConfig.Settings.Processor)
		if err != nil {
			return nil, err
		}
		rule.Outputter, err = f.extractOutputter(ruleConfig.Settings.Outputter)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}
