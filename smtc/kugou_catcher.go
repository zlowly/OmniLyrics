//go:build windows
// +build windows

package smtc

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudmilk/w32uiautomation"
)

const (
	maxFailCount   int32        = 3
	reconnectDelay             = 500 * time.Millisecond
)

var errNotInitialized = errors.New("kugou catcher not initialized")
var errSliderNotFound  = errors.New("slider not found")
var errInvalidValue    = errors.New("invalid value")
var errCoolingDown    = errors.New("cooling down")

type KugouCatcher struct {
	mu       sync.Mutex
	auto    *w32uiautomation.IUIAutomation
	slider  *w32uiautomation.IUIAutomationElement
	rootEl *w32uiautomation.IUIAutomationElement

	valid       bool
	lastSuccess time.Time
	failCount  int32

	mainWindowRegex *regexp.Regexp
}

func NewKugouCatcher() *KugouCatcher {
	auto, err := w32uiautomation.NewUIAutomation()
	if err != nil {
		return nil
	}

	return &KugouCatcher{
		auto:            auto,
		mainWindowRegex: regexp.MustCompile(`酷狗音乐\s*$`),
	}
}

func (k *KugouCatcher) GetPosition() (posMs, durMs int64, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.auto == nil {
		return 0, 0, errNotInitialized
	}

	// 缓存命中且有效
	if k.valid && k.slider != nil {
		posMs, durMs, err = k.readSlider()
		if err == nil {
			k.lastSuccess = time.Now()
			k.failCount = 0
			return posMs, durMs, nil
		}
		k.releaseSlider()
	}

	// 检查是否需要冷却
	if k.failCount >= maxFailCount && time.Since(k.lastSuccess) < reconnectDelay {
		return 0, 0, errCoolingDown
	}

	// 尝试重新定位
	if err := k.findSlider(); err != nil {
		k.failCount++
		k.valid = false
		return 0, 0, err
	}

	// 读取进度
	posMs, durMs, err = k.readSlider()
	if err != nil {
		k.releaseSlider()
		k.valid = false
		k.failCount++
		return 0, 0, err
	}

	k.valid = true
	k.lastSuccess = time.Now()
	k.failCount = 0
	return posMs, durMs, nil
}

func (k *KugouCatcher) readSlider() (posMs, durMs int64, err error) {
	if k.slider == nil {
		return 0, 0, errSliderNotFound
	}

	curValVar, err := k.slider.Get_CurrentPropertyValue(30047)
	if err != nil {
		return 0, 0, err
	}

	maxValVar, err := k.slider.Get_CurrentPropertyValue(30050)
	if err != nil {
		return 0, 0, err
	}

	curF := toFloat64(curValVar.Value())
	maxF := toFloat64(maxValVar.Value())

	if maxF <= 0 {
		return 0, 0, errInvalidValue
	}

	return int64(curF), int64(maxF), nil
}

func (k *KugouCatcher) findSlider() error {
	k.releaseSlider()

	root, err := k.auto.GetRootElement()
	if err != nil {
		return err
	}

	condTrue, err := k.auto.CreateTrueCondition()
	if err != nil {
		root.Release()
		return err
	}
	defer condTrue.Release()

	children, err := root.FindAll(w32uiautomation.TreeScope_Children, condTrue)
	if err != nil {
		root.Release()
		return err
	}
	defer children.Release()

	length, _ := children.Get_Length()
	for i := int32(0); i < length; i++ {
		win, err := children.GetElement(i)
		if err != nil {
			continue
		}

		name, _ := win.Get_CurrentName()
		if !k.mainWindowRegex.MatchString(name) || strings.HasPrefix(name, "桌面歌词") {
			win.Release()
			continue
		}

		nameVar := w32uiautomation.NewVariantString("进度")
		condName, err := k.auto.CreatePropertyCondition(w32uiautomation.UIA_NamePropertyId, nameVar)
		if err != nil {
			win.Release()
			return err
		}
		defer condName.Release()

		slider, err := win.FindFirst(w32uiautomation.TreeScope_Descendants, condName)
		if err != nil || slider == nil {
			win.Release()
			return errSliderNotFound
		}

		k.slider = slider
		k.rootEl = win
		return nil
	}

	return errSliderNotFound
}

func (k *KugouCatcher) releaseSlider() {
	if k.slider != nil {
		k.slider.Release()
		k.slider = nil
	}
	if k.rootEl != nil {
		k.rootEl.Release()
		k.rootEl = nil
	}
}

func (k *KugouCatcher) Release() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.releaseSlider()
	k.auto = nil
}

func toFloat64(v interface{}) float64 {
	switch i := v.(type) {
	case float64:
		return i
	case float32:
		return float64(i)
	case int64:
		return float64(i)
	case int32:
		return float64(i)
	default:
		return 0
	}
}