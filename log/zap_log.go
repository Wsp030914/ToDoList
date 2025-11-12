package log

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

type ZapConfig struct {
	Prefix     string         `yaml:"prefix" mapstructure:"prefix"`
	TimeFormat string         `yaml:"timeFormat" mapstructure:"timeFormat"`
	Level      string         `yaml:"level" mapstructure:"level"`
	Caller     bool           `yaml:"caller" mapstructure:"caller"`
	StackTrace bool           `yaml:"stackTrace" mapstructure:"stackTrace"`
	Writer     string         `yaml:"writer" mapstructure:"writer"`
	Encode     string         `yaml:"encode" mapstructure:"encode"`
	LogFile    *LogFileConfig `yaml:"logFile" mapstructure:"logFile"`
}

type LogFileConfig struct {
	MaxSize  int      `yaml:"maxSize" mapstructure:"maxSize"`
	BackUps  int      `yaml:"backups" mapstructure:"backups"`
	Compress bool     `yaml:"compress" mapstructure:"compress"`
	Output   []string `yaml:"output" mapstructure:"output"`
	Errput   []string `yaml:"errput" mapstructure:"errput"`
}

func LoadZapConfig() (*ZapConfig, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}
	var cfg ZapConfig
	if err := v.UnmarshalKey("zap", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal zap failed: %w", err)
	}

	return &cfg, nil

}

func InitZap(config *ZapConfig) *zap.Logger {
	// 构建编码器
	encoder := zapEncoder(config)
	// 最后获得Core和Options
	subCore, options := tee(config, encoder)
	// 创建Logger
	logger := zap.New(subCore, options...)
	if strings.TrimSpace(config.Prefix) != "" {
		logger = logger.With(zap.String("prefix", config.Prefix))
	}
	return logger
}

func zapEncoder(config *ZapConfig) zapcore.Encoder {
	// 新建一个配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "Time",
		LevelKey:      "Level",
		NameKey:       "Logger",
		CallerKey:     "Caller",
		MessageKey:    "Message",
		StacktraceKey: "StackTrace",
		LineEnding:    zapcore.DefaultLineEnding,
		FunctionKey:   zapcore.OmitKey,
	}

	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(config.TimeFormat)
	// 日志级别大写
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// 秒级时间间隔
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	// 简短的调用者输出
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	// 完整的序列化logger名称
	encoderConfig.EncodeName = zapcore.FullNameEncoder
	// 最终的日志编码 json

	switch strings.ToLower(config.Encode) {
	case "console":
		return zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return zapcore.NewJSONEncoder(encoderConfig)
	}

}

func tee(cfg *ZapConfig, encoder zapcore.Encoder) (zapcore.Core, []zap.Option) {

	al, err := zap.ParseAtomicLevel(strings.ToLower(cfg.Level))
	minLevel := zapcore.InfoLevel
	if err == nil {
		minLevel = al.Level()
	}
	cores := make([]zapcore.Core, 0, 2)
	if cfg.LogFile != nil && len(cfg.LogFile.Output) > 0 {
		infoSink := makeFileSink(cfg.LogFile.Output, cfg.LogFile)
		infoFilter := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
			return l >= minLevel && l < zapcore.ErrorLevel
		})
		infoCore := zapcore.NewCore(encoder, infoSink, infoFilter)
		cores = append(cores, infoCore)
	}

	if cfg.LogFile != nil && len(cfg.LogFile.Errput) > 0 {
		errSink := makeFileSink(cfg.LogFile.Errput, cfg.LogFile)
		start := minLevel
		if start < zapcore.ErrorLevel {
			start = zapcore.ErrorLevel
		}
		errFilter := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
			return l >= start
		})
		errCore := zapcore.NewCore(encoder, errSink, errFilter)
		cores = append(cores, errCore)
	}

	core := zapcore.NewTee(cores...)

	opts := buildOptions(cfg, zapcore.ErrorLevel)
	return core, opts

}

func makeFileSink(paths []string, lf *LogFileConfig) zapcore.WriteSyncer {
	syncers := make([]zapcore.WriteSyncer, 0, len(paths))
	for _, p := range paths {
		lj := &lumberjack.Logger{
			Filename:   p,
			MaxSize:    lf.MaxSize,
			MaxBackups: lf.BackUps,
			Compress:   lf.Compress,
			LocalTime:  true,
		}
		syncers = append(syncers, zapcore.Lock(zapcore.AddSync(lj)))
	}
	return zap.CombineWriteSyncers(syncers...)
}

// 构建Option
func buildOptions(cfg *ZapConfig, levelEnabler zapcore.LevelEnabler) (options []zap.Option) {
	if cfg.Caller {
		options = append(options, zap.AddCaller()) //增加行号
	}

	if cfg.StackTrace {
		options = append(options, zap.AddStacktrace(levelEnabler))
	}
	return
}
