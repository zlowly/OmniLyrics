// PVRenderer - 日式PV字幕展示模式
// 基于 GSAP 和逐字时间戳，生成高质量日式风格字幕动画

class PVRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.params = config?.modeParams?.pv || {};
        this.mainTimeline = null;
        this.currentLineIndex = -1;
        this.isFadingOut = false;
    }

    getDefaultConfig() {
        return {
            template: 'pop_center',
            canvasSize: { width: 1280, height: 720 },
            showNextLine: true,
            baseFontSize: '48px',
            fontFamily: 'Noto Sans JP, Hiragino Kaku Gothic ProN, Yu Gothic, sans-serif',
            fontWeight: 700,
            colors: {
                text: '#f0f0f0',
                fill: '#ffde6e',
                nextText: '#cccccc',
                nextFill: '#aaaaaa'
            },
            effects: {
                glowRange: 0,
                outlineWidth: 0,
                outlineColor: '#000000'
            }
        };
    }

    applyConfig(config) {
        super.applyConfig(config);
        this.params = config?.modeParams?.pv || this.getDefaultConfig();
    }

    // 预设模板配置
    getTemplates() {
        return {
            pop_center: {
                name: '弹跳居中',
                layoutType: 'center',
                baseFontSize: '48px',
                fontFamily: 'Noto Sans JP, Hiragino Kaku Gothic ProN, sans-serif',
                fontWeight: 700,
                charEnter: {
                    from: { y: 18, scale: 0.6, opacity: 0 },
                    to: { y: 0, scale: 1, opacity: 1 },
                    duration: 0.12,
                    ease: 'back.out(0.7)'
                },
                charExit: {
                    to: { scale: 0.7, opacity: 0 },
                    duration: 0.08,
                    ease: 'power2.in'
                },
                fillEffect: 'scaleX',
                fillColor: '#ffde6e',
                fillTiming: 'linear',
                shakeOnHeavy: false,
                shakeIntensity: { x: 3, y: 3, duration: 0.05, repeats: 2 },
                globalFlash: false,
                decorSymbols: false,
                postFX: [],
                nextLineOpacity: 0.45,
                nextLineScale: 0.55
            },
            fast_scatter: {
                name: '快速散射',
                layoutType: 'scatter',
                baseFontSize: '42px',
                fontFamily: 'Noto Sans JP, Hiragino Kaku Gothic ProN, sans-serif',
                fontWeight: 700,
                charEnter: {
                    from: { x: 'random(-100, 100)', y: 'random(-50, 50)', scale: 'random(0.5, 1.2)', rotation: 'random(-15, 15)', opacity: 0 },
                    to: { x: 0, y: 0, scale: 1, rotation: 0, opacity: 1 },
                    duration: 0.25,
                    ease: 'power2.out'
                },
                charExit: {
                    to: { scale: 0.8, opacity: 0 },
                    duration: 0.1,
                    ease: 'power2.in'
                },
                fillEffect: 'scaleX',
                fillColor: '#ffde6e',
                fillTiming: 'linear',
                shakeOnHeavy: false,
                shakeIntensity: { x: 3, y: 3, duration: 0.05, repeats: 2 },
                globalFlash: false,
                decorSymbols: false,
                postFX: [],
                nextLineOpacity: 0.4,
                nextLineScale: 0.5
            },
            cardStack: {
                name: '卡片堆叠',
                layoutType: 'cardStack',
                baseFontSize: '44px',
                fontFamily: 'Noto Sans JP, Hiragino Kaku Gothic ProN, sans-serif',
                fontWeight: 600,
                charEnter: {
                    from: { x: -30, scale: 0.8, opacity: 0 },
                    to: { x: 0, scale: 1, opacity: 1 },
                    duration: 0.15,
                    ease: 'power2.out'
                },
                charExit: {
                    to: { x: 20, scale: 0.9, opacity: 0 },
                    duration: 0.1,
                    ease: 'power2.in'
                },
                fillEffect: 'scaleX',
                fillColor: '#ffde6e',
                fillTiming: 'linear',
                shakeOnHeavy: false,
                shakeIntensity: { x: 3, y: 3, duration: 0.05, repeats: 2 },
                globalFlash: false,
                decorSymbols: false,
                postFX: [],
                nextLineOpacity: 0.4,
                nextLineScale: 0.5,
                cardBg: 'rgba(0, 0, 0, 0.4)',
                cardBorder: '2px solid rgba(255, 255, 255, 0.3)',
                cardPadding: '4px 8px',
                cardBorderRadius: '4px'
            }
        };
    }

    initStyles() {
        this.config = window.configManager?.get() || {};
        const pvConfig = this.getPVConfig();
        const template = this.getCurrentTemplate();

        // 注入样式
        this.injectStyles(template);
    }

    getPVConfig() {
        const defaultConfig = this.getDefaultConfig();
        return { ...defaultConfig, ...this.params };
    }

    getCurrentTemplate() {
        const templates = this.getTemplates();
        const pvConfig = this.getPVConfig();
        return templates[pvConfig.template] || templates.pop_center;
    }

    injectStyles(template) {
        const pvConfig = this.getPVConfig();
        const colors = pvConfig.colors || {};
        const effects = pvConfig.effects || {};

        let styleEl = document.getElementById('pv-styles');
        if (!styleEl) {
            styleEl = document.createElement('style');
            styleEl.id = 'pv-styles';
            document.head.appendChild(styleEl);
        }

        const fillColor = template.fillColor || colors.fill || '#ffde6e';
        const nextFillColor = colors.nextFill || '#aaaaaa';
        const fontSize = template.baseFontSize || pvConfig.baseFontSize || '48px';
        const fontFamily = template.fontFamily || pvConfig.fontFamily;
        const fontWeight = template.fontWeight || pvConfig.fontWeight || 700;

        let cardStyles = '';
        if (template.layoutType === 'cardStack') {
            const cardBg = template.cardBg || 'rgba(0, 0, 0, 0.4)';
            const cardBorder = template.cardBorder || '2px solid rgba(255, 255, 255, 0.3)';
            const cardPadding = template.cardPadding || '4px 8px';
            const cardRadius = template.cardBorderRadius || '4px';

            cardStyles = `
                .pv-char { background: ${cardBg}; border: ${cardBorder}; padding: ${cardPadding}; border-radius: ${cardRadius}; display: inline-block; }
            `;
        }

        styleEl.textContent = `
            .pv-canvas {
                position: relative;
                width: ${pvConfig.canvasSize?.width || 1280}px;
                height: ${pvConfig.canvasSize?.height || 720}px;
                background: transparent;
                overflow: hidden;
            }
            .pv-subtitle-layer {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                background: transparent;
                pointer-events: none;
            }
            .pv-line {
                position: absolute;
                bottom: 12%;
                left: 0;
                right: 0;
                display: flex;
                justify-content: center;
                align-items: baseline;
                flex-wrap: nowrap;
                gap: 0.08em;
                white-space: nowrap;
                font-size: ${fontSize};
                font-weight: ${fontWeight};
                font-family: ${fontFamily};
                letter-spacing: 0.02em;
                text-shadow: 2px 2px 6px rgba(0,0,0,0.3);
                background: transparent;
            }
            .pv-char {
                position: relative;
                display: inline-block;
                transform-origin: center center;
                will-change: transform, opacity;
            }
            .pv-char-text {
                display: block;
                position: relative;
                color: ${colors.text || '#f0f0f0'};
                transition: color 0.1s ease;
                text-shadow: 0px 0px 2px #000, 1px 1px 0 #000;
            }
            .pv-char-fill {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                background: ${fillColor};
                transform-origin: left center;
                transform: scaleX(0);
                mix-blend-mode: overlay;
                pointer-events: none;
                z-index: 1;
            }
            .pv-line.next-line {
                bottom: 4%;
                font-size: ${template.nextLineScale || 0.55}em;
                opacity: ${template.nextLineOpacity || 0.45};
                filter: blur(0.4px);
                text-shadow: none;
            }
            .pv-line.next-line .pv-char-text {
                color: ${colors.nextText || '#cccccc'};
                text-shadow: 0 0 1px black;
            }
            .pv-line.next-line .pv-char-fill {
                background: ${nextFillColor};
                mix-blend-mode: normal;
                opacity: 0.6;
            }
            ${cardStyles}
        `;
    }

    // 构建 DOM 结构
    // lrcData: 原始歌词数据数组，每个元素包含 text、time、endTime、words 等
    buildDOM(lrcData) {
        if (!this.container || !lrcData || !lrcData.length) return [];

        this.container.innerHTML = '';
        const lineElements = [];
        const pvConfig = this.getPVConfig();

        lrcData.forEach((line, lineIdx) => {
            const lineDiv = document.createElement('div');
            lineDiv.className = 'pv-line';
            lineDiv.setAttribute('data-line-idx', lineIdx);

            // 构建字符元素
            const words = line.words || [];
            if (words.length > 0) {
                // 有逐字信息，按单词/字符显示
                words.forEach((word, wordIdx) => {
                    const charSpan = document.createElement('span');
                    charSpan.className = 'pv-char';
                    charSpan.setAttribute('data-char-idx', `${lineIdx}_${wordIdx}`);

                    const textSpan = document.createElement('span');
                    textSpan.className = 'pv-char-text';
                    textSpan.textContent = word.text;

                    const fillSpan = document.createElement('span');
                    fillSpan.className = 'pv-char-fill';

                    charSpan.appendChild(textSpan);
                    charSpan.appendChild(fillSpan);
                    lineDiv.appendChild(charSpan);
                });
            } else {
                // 无逐字信息，按字符显示
                const text = line.text || '';
                const chars = text.split('').filter(c => c.trim());

                chars.forEach((char, charIdx) => {
                    const charSpan = document.createElement('span');
                    charSpan.className = 'pv-char';
                    charSpan.setAttribute('data-char-idx', `${lineIdx}_${charIdx}`);

                    const textSpan = document.createElement('span');
                    textSpan.className = 'pv-char-text';
                    textSpan.textContent = char;

                    const fillSpan = document.createElement('span');
                    fillSpan.className = 'pv-char-fill';

                    charSpan.appendChild(textSpan);
                    charSpan.appendChild(fillSpan);
                    lineDiv.appendChild(charSpan);
                });
            }

            this.container.appendChild(lineDiv);
            lineElements.push(lineDiv);
        });

        // 初始隐藏所有行
        lineElements.forEach(line => {
            line.style.opacity = '0';
            line.style.visibility = 'hidden';
        });

        return lineElements;
    }

    // 构建 GSAP Timeline
    // lrcData: 原始歌词数据，直接使用其中的 time/endTime（毫秒）
    buildTimeline(lrcData, lineDivs, template) {
        const pvConfig = this.getPVConfig();
        const tl = gsap.timeline({ paused: true });
        const animLog = [];  // 按行分组的动画日志

        // 辅助记录函数
        function logAnim(lineIdx, type, startTime, duration, targetDesc) {
            if (!animLog[lineIdx]) {
                animLog[lineIdx] = [];
            }
            animLog[lineIdx].push({
                type,
                startTime: parseFloat(startTime.toFixed(3)),
                endTime: parseFloat((startTime + duration).toFixed(3)),
                duration: parseFloat(duration.toFixed(3)),
                targetDesc
            });
        }

        if (!lrcData || !lrcData.length) return tl;

        // 计算每行的时间范围（毫秒 → 秒，GSAP使用秒作为时间单位）
        const lineTimeRanges = lrcData.map(line => ({
            start: line.time / 1000,
            end: (line.endTime || line.time + 2000) / 1000
        }));

        // 整行显隐动画
        lrcData.forEach((line, lineIdx) => {
            const lineStart = lineTimeRanges[lineIdx].start;
            const lineEnd = lineTimeRanges[lineIdx].end;
            const lineDiv = lineDivs[lineIdx];

            if (!lineDiv) return;

            // 句子开始时显示行
            tl.set(lineDiv, { visibility: 'visible', opacity: 0, scale: 0.96 }, lineStart);
            logAnim(lineIdx, '整行显示', lineStart, 0, `行 ${lineIdx}`);

            tl.to(lineDiv, { opacity: 1, scale: 1, duration: 0.12, ease: 'power2.out' }, lineStart);
            logAnim(lineIdx, '整行淡入', lineStart, 0.12, `行 ${lineIdx}`);

            // 句子末尾淡出
            tl.to(lineDiv, { opacity: 0, duration: 0.1, ease: 'power1.in' }, lineEnd);
            logAnim(lineIdx, '整行淡出', lineEnd, 0.1, `行 ${lineIdx}`);

            tl.set(lineDiv, { visibility: 'hidden' }, lineEnd + 0.05);
            logAnim(lineIdx, '整行隐藏', lineEnd + 0.05, 0, `行 ${lineIdx}`);
        });

        // 下一行预览效果
        const showNextLine = pvConfig.showNextLine !== false;
        if (showNextLine) {
            for (let i = 0; i < lrcData.length - 1; i++) {
                const currentLineEnd = lineTimeRanges[i].end;
                const nextLineDiv = lineDivs[i + 1];

                if (!nextLineDiv) continue;

                // 在当前行结束后显示下一行预览
                tl.set(nextLineDiv, { visibility: 'visible', opacity: 0.4, scale: 0.85 }, currentLineEnd - 0.2);
                logAnim(i + 1, '下一行预览显示', currentLineEnd - 0.2, 0, `行 ${i + 1}`);

                tl.to(nextLineDiv, { opacity: 0.55, duration: 0.2 }, currentLineEnd - 0.2);
                logAnim(i + 1, '下一行预览淡入', currentLineEnd - 0.2, 0.2, `行 ${i + 1}`);

                // 下一句真正开始时恢复正常样式
                const nextLineStart = lineTimeRanges[i + 1].start;
                tl.to(nextLineDiv, { opacity: 1, scale: 1, duration: 0.15, clearProps: 'all' }, nextLineStart);
                logAnim(i + 1, '下一行恢复样式', nextLineStart, 0.15, `行 ${i + 1}`);
            }
        }

        // 逐字入场 + 卡拉OK填充动画
        lrcData.forEach((line, lineIdx) => {
            const lineDiv = lineDivs[lineIdx];
            if (!lineDiv) return;

            const charElements = lineDiv.querySelectorAll('.pv-char');
            const words = line.words || [];

            if (words.length > 0) {
                // 有逐字时间戳
                words.forEach((word, wordIdx) => {
                    // 毫秒 → 秒（GSAP Timeline时间单位）
                    const wordStart = word.time / 1000;
                    const wordEnd = wordIdx < words.length - 1
                        ? words[wordIdx + 1].time / 1000
                        : (line.endTime || line.time + 2000) / 1000;

                    if (wordIdx >= charElements.length) return;

                    const charEl = charElements[wordIdx];
                    const fillEl = charEl.querySelector('.pv-char-fill');

                    // 入场动画
                    const enterConfig = template.charEnter || {};
                    const from = { ...enterConfig.from };
                    const to = { ...enterConfig.to };
                    const duration = enterConfig.duration || 0.12;
                    const ease = enterConfig.ease || 'back.out(0.7)';

                    tl.fromTo(charEl, from, { ...to, duration, ease }, wordStart);
                    logAnim(lineIdx, '字符入场', wordStart, duration, `"${word.text}"`);

                    // 卡拉OK填充动画
                    let fillCompletionTime = wordEnd;
                    if (fillEl && template.fillEffect !== 'none') {
                        const fillDuration = wordEnd - wordStart;
                        const fillTiming = template.fillTiming || 'linear';

                        tl.fromTo(fillEl,
                            { scaleX: 0 },
                            { scaleX: 1, duration: fillDuration, ease: fillTiming },
                            wordStart
                        );
                        logAnim(lineIdx, '填充', wordStart, fillDuration, `"${word.text}"填充层`);
                        fillCompletionTime = wordStart + fillDuration;
                    }

                    // 退场动画（在填充完成后开始）
                    const exitConfig = template.charExit || {};
                    if (exitConfig.to) {
                        const exitDuration = exitConfig.duration || 0.08;
                        const exitEase = exitConfig.ease || 'power2.in';
                        const exitStart = fillCompletionTime + 0.05;
                        tl.to(charEl, { ...exitConfig.to, duration: exitDuration, ease: exitEase }, exitStart);
                        logAnim(lineIdx, '字符退场', exitStart, exitDuration, `"${word.text}"`);
                    }

                    // 长音字震动效果
                    if (template.shakeOnHeavy) {
                        const avgDuration = (words.reduce((sum, w, idx) => {
                            const ws = w.time / 1000;
                            const we = idx < words.length - 1 ? words[idx + 1].time / 1000 : (line.endTime || line.time + 2000) / 1000;
                            return sum + (we - ws);
                        }, 0) / words.length);

                        if ((wordEnd - wordStart) > avgDuration * 1.5) {
                            const shakeIntensity = template.shakeIntensity || { x: 3, y: 3, duration: 0.05, repeats: 2 };
                            const shakeDuration = shakeIntensity.duration || 0.05;
                            tl.to(charEl, {
                                x: `random(-${shakeIntensity.x}, ${shakeIntensity.x})`,
                                y: `random(-${shakeIntensity.y}, ${shakeIntensity.y})`,
                                duration: shakeDuration,
                                repeat: shakeIntensity.repeats,
                                yoyo: true,
                                ease: 'none'
                            }, wordStart);
                            logAnim(lineIdx, '长音震动', wordStart, shakeDuration, `"${word.text}"`);
                        }
                    }
                });
            } else {
                // 无逐字时间戳，按字符均匀分配时间
                const text = line.text || '';
                const chars = text.split('').filter(c => c.trim());
                // 毫秒 → 秒
                const lineStart = line.time / 1000;
                const lineEnd = (line.endTime || line.time + 2000) / 1000;

                chars.forEach((char, charIdx) => {
                    // 均匀分配每个字符的时间段
                    const charStart = lineStart + (charIdx / Math.max(chars.length, 1)) * (lineEnd - lineStart);
                    const charEnd = charIdx < chars.length - 1
                        ? lineStart + ((charIdx + 1) / Math.max(chars.length, 1)) * (lineEnd - lineStart)
                        : lineEnd;

                    if (charIdx >= charElements.length) return;

                    const charEl = charElements[charIdx];
                    const fillEl = charEl.querySelector('.pv-char-fill');

                    // 入场动画
                    const enterConfig = template.charEnter || {};
                    const from = { ...enterConfig.from };
                    const to = { ...enterConfig.to };
                    const duration = enterConfig.duration || 0.12;
                    const ease = enterConfig.ease || 'back.out(0.7)';

                    tl.fromTo(charEl, from, { ...to, duration, ease }, charStart);
                    logAnim(lineIdx, '字符入场', charStart, duration, `"${char}"`);

                    // 卡拉OK填充动画
                    let fillCompletionTime = charEnd;
                    if (fillEl && template.fillEffect !== 'none') {
                        const fillDuration = charEnd - charStart;
                        const fillTiming = template.fillTiming || 'linear';

                        tl.fromTo(fillEl,
                            { scaleX: 0 },
                            { scaleX: 1, duration: fillDuration, ease: fillTiming },
                            charStart
                        );
                        logAnim(lineIdx, '填充', charStart, fillDuration, `"${char}"填充层`);
                        fillCompletionTime = charStart + fillDuration;
                    }

                    // 退场动画（在填充完成后开始）
                    const exitConfig = template.charExit || {};
                    if (exitConfig.to) {
                        const exitDuration = exitConfig.duration || 0.08;
                        const exitEase = exitConfig.ease || 'power2.in';
                        const exitStart = fillCompletionTime + 0.05;
                        tl.to(charEl, { ...exitConfig.to, duration: exitDuration, ease: exitEase }, exitStart);
                        logAnim(lineIdx, '字符退场', exitStart, exitDuration, `"${char}"`);
                    }
                });
            }
        });

        // 将日志挂载到全局，方便控制台查看
        window.__gsapAnimLog = animLog;
        window.printTimeline = function() {
            console.log('=== PV模式 Timeline 动画记录 ===');
            animLog.forEach((lineAnims, lineIdx) => {
                if (lineAnims && lineAnims.length > 0) {
                    console.group(`第 ${lineIdx} 行 (共${lineAnims.length}个动画)`);
                    console.table(lineAnims);
                    console.groupEnd();
                }
            });
            console.log(`总动画数: ${animLog.flat().length}`);
            console.log('时间线总时长:', tl.duration().toFixed(3));
        };

        return tl;
    }

    render(frameData) {
        const motion = window.motion;
        if (!motion?.lrcData?.length) {
            this.renderPlaceholder();
            return;
        }

        const line = motion.lrcData[frameData.currentIndex] || motion.lrcData[0];
        if (!line) return;

        const { position, isPlaying, currentIndex, isSongEnded } = frameData;

        // 播放结束淡出
        if (isSongEnded && !isPlaying) {
            if (!this.isFadingOut && this.container) {
                this.isFadingOut = true;
                window.gsap.to(this.container, {
                    opacity: 0,
                    duration: 0.4,
                    ease: 'power2.out'
                });
            }
            return;
        }

        // 恢复显示
        if (this.isFadingOut && this.container) {
            this.isFadingOut = false;
            window.gsap.to(this.container, { opacity: 1, duration: 0.2 });
        }

        // 换行检测
        if (this.lastLineIndex !== currentIndex) {
            this.lastLineIndex = currentIndex;
            this.reset();

            if (motion.lrcData.length === 0) return;

            const template = this.getCurrentTemplate();
            const lineDivs = this.buildDOM(motion.lrcData);
            this.mainTimeline = this.buildTimeline(motion.lrcData, lineDivs, template);

            // 同步到当前位置
            if (this.mainTimeline && position > 0) {
                // 毫秒 → 秒
                const targetTime = position / 1000;

                // 找到当前行的开始时间
                let currentLineStart = 0;
                for (let i = 0; i < motion.lrcData.length; i++) {
                    const line = motion.lrcData[i];
                    // 毫秒 → 秒用于比较
                    if (targetTime >= line.time / 1000) {
                        currentLineStart = line.time / 1000;
                    }
                }

                if (targetTime > currentLineStart && targetTime < this.mainTimeline.duration()) {
                    this.mainTimeline.seek(targetTime);
                }

                if (isPlaying) {
                    this.mainTimeline.play();
                } else {
                    this.mainTimeline.pause();
                }
            }
        }

        // 处理播放状态
        if (this.mainTimeline) {
            const currentPos = frameData.position / 1000;
            const duration = this.mainTimeline.duration();

            // 暂停状态
            if (!isPlaying) {
                if (currentPos >= 0 && currentPos <= duration) {
                    this.mainTimeline.seek(currentPos);
                    this.mainTimeline.pause();
                }
            } else {
                // 播放状态
                if (this.mainTimeline.paused()) {
                    this.mainTimeline.play();
                }
            }
        }
    }

    renderPlaceholder() {
        if (this.container) {
            this.container.innerHTML = '<span class="shadow">等待播放...</span>';
        }
    }

    clear() {
        super.clear();
        if (this.mainTimeline) {
            this.mainTimeline.kill();
            this.mainTimeline = null;
        }
        this.currentLineIndex = -1;
    }

    reset() {
        this.clear();
        if (this.container) {
            this.container.innerHTML = '';
        }
    }

    destroy() {
        this.clear();
        const styleEl = document.getElementById('pv-styles');
        if (styleEl) {
            styleEl.remove();
        }
        super.destroy();
    }
}

window.PVRenderer = PVRenderer;