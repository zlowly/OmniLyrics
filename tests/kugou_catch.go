package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudmilk/w32uiautomation"
)

func main() {
	mainWindowRegex := regexp.MustCompile(`酷狗音乐\s*$`)
	auto, _ := w32uiautomation.NewUIAutomation()

	var cachedSlider *w32uiautomation.IUIAutomationElement

	fmt.Println("OmniLyrics Engine - 极速模式已启动")

	for {
		// 1. 缓存检查：如果已经拿到了进度条句柄，直接读取，不走任何搜索逻辑
		if cachedSlider != nil {
			curValVar, errCur := cachedSlider.Get_CurrentPropertyValue(30047)
			maxValVar, errMax := cachedSlider.Get_CurrentPropertyValue(30050)

			if errCur == nil && errMax == nil {
				curF := toFloat64(curValVar.Value())
				maxF := toFloat64(maxValVar.Value())
				if maxF > 0 {
					fmt.Printf("\r进度: %02d:%02d / %02d:%02d (%.0f)      ",
						int(curF/100)/60, int(curF/100)%60,
						int(maxF/100)/60, int(maxF/100)%60, curF)
				}
				time.Sleep(100 * time.Millisecond) // 高频抓取，几乎不占 CPU
				continue
			}
			// 如果报错，说明酷狗切歌了或者 UI 刷新了，释放缓存重新找
			fmt.Println("\n[!] 进度条句柄失效，正在重连...")
			cachedSlider.Release()
			cachedSlider = nil
		}

		// 2. 搜索逻辑：利用NewVariantString
		fmt.Print("\r正在定位酷狗主窗口...          ")
		root, _ := auto.GetRootElement()

		// 找到主窗口
		condTrue, _ := auto.CreateTrueCondition()
		children, _ := root.FindAll(w32uiautomation.TreeScope_Children, condTrue)

		if children != nil {
			count, _ := children.Get_Length()
			for i := int32(0); i < count; i++ {
				win, _ := children.GetElement(i)
				name, _ := win.Get_CurrentName()

				if mainWindowRegex.MatchString(name) && !strings.HasPrefix(name, "桌面歌词") {
					nameVar := w32uiautomation.NewVariantString("进度")
					condName, _ := auto.CreatePropertyCondition(w32uiautomation.UIA_NamePropertyId, nameVar)

					// 直接查找
					slider, err := win.FindFirst(w32uiautomation.TreeScope_Descendants, condName)
					if err == nil && slider != nil {
						cachedSlider = slider // 存入缓存
						fmt.Println("\n[OK] 捕获成功！已锁定进度条句柄。")
						break
					}
					condName.Release()
				}
				win.Release()
			}
			children.Release()
		}
		root.Release()

		if cachedSlider == nil {
			time.Sleep(1 * time.Second) // 没找到时降低频率
		}
	}
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
