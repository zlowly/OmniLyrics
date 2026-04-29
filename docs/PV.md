# OmniLyrics PV字幕模式需求文档

## 1. 项目概述

### 1.1 功能定位
基于 GSAP 和逐字时间戳，生成高质量日式风格字幕动画，输出带透明背景的 DOM 元素层，供第三方程序（如 OBS、视频编辑软件、其他渲染引擎）叠加使用。

### 1.2 核心目标
- 透明背景容器，尺寸可配置
- 每个字符按时间线出现，使用 GSAP 驱动动画
- 支持同时显示多行（当前句 + 下一句预览）
- 可独立导出使用，提供 init() 和 destroy() 方法

### 1.3 不包含功能
- 背景光效、视频合成
- 音频节拍分析、实时振幅检测
- 多层级效果管理（背景/装饰层分离）

---

## 2. 技术依赖

| 依赖 | 版本要求 | 用途 |
|------|---------|------|
| GSAP | >= 3.12 | 动画驱动 |
| 原生 DOM API | - | DOM 操作 |

---

## 3. 数据格式

### 3.1 输入数据 (lrcData)
项目现有的歌词数据结构：

```javascript
[
    {
        time: 3000,        // 行开始时间 (毫秒)
        text: "歌词内容",
        words: [            // 逐字时间戳（可选）
            { time: 3000, text: "歌" },
            { time: 3300, text: "词" },
            ...
        ],
        endTime: 5000       // 行结束时间 (毫秒)
    },
    ...
]
```

### 3.2 内部转换格式
PV 模块内部使用的格式：

```javascript
{
    text: "歌",           // 字符文本
    start: 3.0,          // 开始时间 (秒)
    end: 3.3,            // 结束时间 (秒)
    lineId: 0,           // 行索引
    isHeavy: false        // 是否为长音/重音字
}
```

---

## 4. 预设模板系统

### 4.1 模板配置结构

```javascript
templates: {
    '<模板名称>': {
        name: '<显示名称>',
        layoutType: 'center',      // 排版方式
        baseFontSize: '48px',     // 基准字号
        fontFamily: 'Noto Sans JP, Hiragino Kaku Gothic ProN, sans-serif',
        charEnter: {              // 入场动画参数
            from: { y: 18, scale: 0.6, opacity: 0 },
            to: { y: 0, scale: 1, opacity: 1 },
            duration: 0.12,
            ease: 'back.out(0.7)'
        },
        charExit: {              // 退场动画参数
            to: { scale: 0.7, opacity: 0 },
            duration: 0.08,
            ease: 'power2.in'
        },
        fillEffect: 'scaleX',    // 卡拉OK填充方式: scaleX | clipPath | colorShift | none
        fillTiming: 'linear',   // 填充速度曲线
        shakeOnHeavy: false,    // 长音字是否震动
        shakeIntensity: { x: 3, y: 3, duration: 0.05, repeats: 2 },
        globalFlash: false,     // 句首/重音是否闪白
        decorSymbols: false,     // 是否添加装饰符号
        postFX: []            // 后期处理链
    }
}
```

### 4.2 排版方式 (layoutType)

| 类型 | 说明 | 实现方式 |
|------|------|----------|
| center | 居中 | Flexbox 居中，每个字符按自然顺序排列 |
| scatter | 散射 | 绝对定位 + 避碰算法，字符随机倾斜、大小变化 |
| cardStack | 卡片 | 字符带背景卡片（圆角、半透明底、左框线），堆叠显示 |

### 4.3 填充效果 (fillEffect)

| 类型 | 说明 |
|------|------|
| scaleX | 填充层 scaleX 从 0→1 |
| clipPath | 填充层使用 clip-path 裁剪 |
| colorShift | 文字颜色渐变 |
| none | 无填充效果 |

### 4.4 预定义模板

#### 4.4.1 pop_center (弹跳居中)
- 排版：center (Flexbox 居中)
- 入场：从下方弹跳 + 缩放 (back.out)
- 填充：scaleX 亮黄色
- 风格：日式 PV 常用，节奏感强

#### 4.4.2 fast_scatter (快速散射)
- 排版：scatter (绝对定位避碰)
- 入场：随机位置飞入 + 旋转
- 填充：scaleX
- 风格：活泼、现代感

#### 4.4.3 cardStack (卡片堆叠)
- 排版：cardStack (带背景卡片)
- 入场：从侧边滑入 + 堆叠效果
- 填充：scaleX
- 风格：沉稳、阅读性强

---

## 5. 核心功能模块

### 5.1 数据转换层 (convertLrcData)
将 lrcData 转换为 PV 内部格式：

```javascript
function convertLrcData(lrcData) {
    // 1. 遍历每行，提取逐字时间戳
    // 2. 计算每个字的结束时间
    // 3. 标记长音字 (duration > avg * 1.5)
    // 4. 返回转换后的数组 + 分组信息
}
```

### 5.2 排版引擎 (LayoutManager)
根据 layoutType 为每个字符生成位置：

- **center**: 使用 Flexbox 居中，简单高效
- **scatter**: 绝对定位 + 网格避碰算法
- **cardStack**: 计算堆叠偏移量

### 5.3 时间线构建器 (TimelineBuilder)
接收 words[] + 模板配置 + 布局结果，生成 GSAP Timeline：

- 每个字符的入场动画（从 template.charEnter）
- 每个字符的卡拉OK填充动画（从 template.fillEffect）
- 每个字符的退场动画（从 template.charExit）
- 整行显隐控制
- 下一行预览效果

### 5.4 后期处理链 (PostFX)
作用于整个字幕层容器：

| 效果 | 说明 |
|------|------|
| GlobalShake | 全屏震动 |
| Flash | 闪白（半透明白层脉冲）|
| Chromatic | 色差（text-shadow 模拟）|
| Glitch | 故障效果 |
| ScalePulse | 缩放脉冲 |

---

## 6. 配置结构

### 6.1 独立配置（pv）
每个展示模式有独立的字体配置：

```javascript
pv: {
    mode: 'pv',                    // 模式标识
    template: 'pop_center',         // 当前模板
    canvasSize: {                  // 画布尺寸
        width: 1280,
        height: 720
    },
    showNextLine: true,            // 显示下一句预览
    baseFontSize: '48px',          // 基准字号
    fontFamily: 'Noto Sans JP, Hiragino Kaku Gothic ProN, sans-serif',
    fontWeight: 700,
    colors: {
        text: '#f0f0f0',         // 文字颜色
        fill: '#ffde6e',           // 填充颜色
        nextText: '#cccccc',        // 下一句文字颜色
        nextFill: '#aaaaaa'        // 下一句填充颜色
    },
    effects: {
        glowRange: 0,
        outlineWidth: 0,
        outlineColor: '#000000'
    },
    // 模板参数微调（可选）
    templateParams: {}
}
```

---

## 7. 集成方式

### 7.1 与 RendererManager 集成
在 renderers/index.js 中添加：

```javascript
case 'pv':
    this.currentRenderer = new window.PVRenderer(this.container, this.stage, this.config);
    break;
```

### 7.2 独立导出 API

```javascript
// 初始化
window.PVLyrics.init(lrcData, containerId, options);

// 销毁
window.PVLyrics.destroy();

// options 可选参数
{
    template: 'pop_center',
    canvasSize: { width: 1280, height: 720 },
    baseFontSize: '48px',
    showNextLine: true
}
```

---

## 8. 实现分阶段

### Phase 1: 基础功能
- PVRenderer 主类
- 数据转换层
- buildDOM() / buildTimeline() 核心逻辑
- 与 RendererManager 集成
- 仅支持 center 排版

### Phase 2: 模板系统
- 预设 3 种模板配置
- 模板切换逻辑

### Phase 3: 排版扩展
- scatter 散射排版
- cardStack 卡片排版

### Phase 4: 后期处理
- PostFX 后期处理链

### Phase 5: 独立导出
- window.PVLyrics API

---

## 9. 测试参考

- 测试文件：`tests/pv_test.html`
- 该文件保留作为独立演示和对比使用
- 实现逻辑参考该文件的 buildDOM() 和 buildTimeline()