package rofi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Option struct {
	Category string
	Cmd      string
	Icon     string
	Name     string
	Value    string
}

type Options []Option

type Value struct {
	Cmd      string
	Value    string
	Modifier int
}

var verbosity = 0
var history = ""

const maxHistoryCount = 5

func init() {
	fmt.Println("\000no-custom\x1ftrue")

	if v := os.Getenv("ROFI_DEBUG"); v != "" {
		if num, err := strconv.Atoi(v); err == nil {
			Debug(num)
		}
	}
}

func (o Option) Print() {
	if o.Name == "" {
		if verbosity >= 5 {
			log.Println("Option was empty")
		}
		return

	}

	str := o.Name
	if o.Category != "" {
		str = fmt.Sprintf("%s (%s)", str, o.Category)
	}

	if o.Cmd != "" {
		str = fmt.Sprintf("%s\000info\x1f%s:%s", str, o.Cmd, o.Value)
	} else {
		str = fmt.Sprintf("%s\000info\x1f%s", str, o.Value)
	}

	if o.Icon != "" {
		str = fmt.Sprintf("%s\x1ficon\x1f%s", str, o.Icon)
	}

	if verbosity >= 5 {
		log.Println("Option:", str)
	}

	fmt.Println(str)
}

func (opts Options) Sort() {
	sort.Slice(opts, func(a, b int) bool {
		return strings.ToLower(opts[a].Name) < strings.ToLower(opts[b].Name)
	})
}

func (opts Options) PrintAll() {
	if history != "" {
		opts.PrioritizeHistory(history)
	}

	for _, o := range opts {
		o.Print()
	}
}

func (o *Options) PrioritizeHistory(namespace string) {
	opts := *o
	cache, err := getCachePath(namespace)
	if err != nil {
		log.Printf("Error while finding cache: %s\n", err)
		return
	}
	f, err := os.OpenFile(cache, os.O_RDONLY, 0666)
	if err != nil {
		log.Printf("Error while opening cache: %s\n", err)
		return
	}
	defer f.Close()

	history, err := readHistory(f)
	if err != nil {
		log.Printf("Error while reading history: %s", err)
	}

	prio := []Option{}

	for _, h := range history {
		for i, opt := range opts {
			if h == opt.Value {
				opts = append(opts[:i], opts[i+1:]...)
				prio = append(prio, opt)
			}
		}
	}

	*o = append(prio, opts...)
}

func SetPrompt(prompt string) {
	fmt.Printf("\000prompt\x1f%s\n", prompt)
}

func SetMessage(message string) {
	fmt.Printf("\000message\x1f%s\n", message)
}

func GetValue() *Value {
	if v := os.Getenv("ROFI_INFO"); v != "" {
		val := Value{}
		if s := GetState(); s >= 10 {
			val.Modifier = s - 9
		}

		values := strings.SplitN(v, ":", 2)
		if len(values) == 2 {
			val.Cmd = values[0]
			val.Value = values[1]
		} else {
			val.Value = values[0]
		}

		return &val
	}

	return nil
}

func EnableHotkeys() {
	if verbosity > 3 {
		log.Println("Enabled hotkeys")
	}
	fmt.Println("\000use-hot-keys\x1ftrue")
}

func GetState() int {
	num, _ := strconv.Atoi(os.Getenv("ROFI_RETV"))
	return num
}

/*
	1-5, 0 means off
*/
func Debug(verbosityLevel int) {
	if verbosityLevel > 0 {
		f, err := os.OpenFile("rofi-debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return
		}
		log.Default().SetOutput(f)
		log.SetFlags(0)
		log.Printf("\n---------------------------------------------------\n\n")
		log.SetFlags(log.LstdFlags)

		verbosity = verbosityLevel
	}
}

func EnableCustom() {
	if verbosity > 3 {
		log.Println("Enabled custom entries")
	}
	fmt.Println("\000no-custom\x1ffalse")
}

func GetVerbosityLevel() int {
	return verbosity
}

func UseHistory(namespace string) {
	history = namespace
}

func getCachePath(namespace string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return path.Join(cache, "/rofi/", namespace+".json"), nil
}

func readHistory(f *os.File) ([]string, error) {
	var content []string

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return content, err
	}

	err = json.Unmarshal(b, &content)

	return content, err
}

func writeHistory(f *os.File, content []string) error {
	b, err := json.MarshalIndent(content, "", "  ")

	if err != nil {
		return fmt.Errorf("error while marshalling history: %w", err)
	}

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("error while clearing history: %w", err)
	}

	if _, err := f.WriteAt(b, 0); err != nil {
		return fmt.Errorf("error while writing history: %w", err)
	}

	return nil
}

func SaveToHistory(namespace, value string) {
	cache, err := getCachePath(namespace)
	if err != nil {
		log.Printf("Error while finding cache: %s\n", err)
		return
	}

	if err := os.MkdirAll(path.Dir(cache), os.ModePerm); err != nil {
		log.Printf("Error while creating path: %s\n", err)
		return
	}

	f, err := os.OpenFile(cache, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Error while opening cache: %s\n", err)
		return
	}

	defer f.Close()

	history, err := readHistory(f)
	if err != nil {
		log.Printf("Error while reading history: %s\n", err)
	}

	// shifting
	for i, val := range history {
		if val == value {
			history = append(history[:i], history[i+1:]...)
		}
	}

	nextHistory := []string{value}
	if len(history) >= maxHistoryCount {
		nextHistory = append(nextHistory, history[:maxHistoryCount-1]...)
	} else {
		nextHistory = append(nextHistory, history...)
	}

	if err := writeHistory(f, nextHistory); err != nil {
		log.Printf("Error while saving history: %s\n", err)
	}
}
