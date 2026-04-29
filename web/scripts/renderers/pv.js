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

    // 将 lrcData 转换为 PV 内部格式
    convertLrcData(lrcData) {
        if (!lrcData || !lrcData.length) return { flat: [], grouped: [] };

        const flat = [];
        const grouped = [];
        let globalIndex = 0;

        lrcData.forEach((line, lineId) => {
            const lineWords = [];
            const text = line.text || '';
            const words = line.words || [];

            if (words.length > 1) {
                // 有逐字时间戳
                words.forEach((word, idx) => {
                    const start = word.time / 1000; // 转换为秒
                    const end = idx < words.length - 1
                        ? words[idx + 1].time / 1000
                        : (line.endTime || line.time + 2000) / 1000;

                    flat.push({
                        text: word.text,
                        start,
                        end,
                        lineId,
                        globalIndex: globalIndex++,
                        isHeavy: false
                    });
                    lineWords.push({
                        text: word.text,
                        start,
                        end,
                        lineId,
                        globalIndex: globalIndex - 1,
                        isHeavy: false
                    });
                });
            } else if (text) {
                // 无逐字时间戳，整行处理
                const start = line.time / 1000;
                const end = (line.endTime || line.time + 2000) / 1000;
                const chars = text.split('').filter(c => c.trim());

                chars.forEach((char, idx) => {
                    const charStart = start + (idx / Math.max(chars.length, 1)) * (end - start);
                    const charEnd = idx < chars.length - 1
                        ? start + ((idx + 1) / Math.max(chars.length, 1)) * (end - start)
                        : end;

                    flat.push({
                        text: char,
                        start: charStart,
                        end: charEnd,
                        lineId,
                        globalIndex: globalIndex++,
                        isHeavy: false
                    });
                    lineWords.push({
                        text: char,
                        start: charStart,
                        end: charEnd,
                        lineId,
                        globalIndex: globalIndex - 1,
                        isHeavy: false
                    });
                });
            }

            if (lineWords.length > 0) {
                grouped.push(lineWords);
            }
        });

        // 标记长音字
        if (flat.length > 0) {
            const avgDuration = flat.reduce((sum, w) => sum + (w.end - w.start), 0) / flat.length;
            flat.forEach(w => {
                w.isHeavy = (w.end - w.start) > avgDuration * 1.5;
            });
        }

        return { flat, grouped };
    }

    // 构建 DOM 结构
    buildDOM(wordsGrouped) {
        if (!this.container) return [];

        this.container.innerHTML = '';
        const lineElements = [];
        const pvConfig = this.getPVConfig();
        const showNextLine = pvConfig.showNextLine !== false;

        wordsGrouped.forEach((lineWords, lineIdx) => {
            const lineDiv = document.createElement('div');
            lineDiv.className = 'pv-line';
            lineDiv.setAttribute('data-line-idx', lineIdx);

            // 构建字符元素
            lineWords.forEach((word, wordIdx) => {
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
    buildTimeline(wordsFlat, wordsGrouped, lineDivs, template) {
        const pvConfig = this.getPVConfig();
        const tl = gsap.timeline({ paused: true });

        // 计算每行的时间范围
        const lineTimeRanges = wordsGrouped.map(lineWords => ({
            start: lineWords[0].start,
            end: lineWords[lineWords.length - 1].end
        }));

        // 整行显隐动画
        wordsGrouped.forEach((lineWords, lineIdx) => {
            const lineStart = lineTimeRanges[lineIdx].start;
            const lineEnd = lineTimeRanges[lineIdx].end;
            const lineDiv = lineDivs[lineIdx];

            // 句子开始时显示行
            tl.set(lineDiv, { visibility: 'visible', opacity: 0, scale: 0.96 }, lineStart);
            tl.to(lineDiv, { opacity: 1, scale: 1, duration: 0.12, ease: 'power2.out' }, lineStart);

            // 句子末尾淡出
            tl.to(lineDiv, { opacity: 0, duration: 0.1, ease: 'power1.in' }, lineEnd);
            tl.set(lineDiv, { visibility: 'hidden' }, lineEnd + 0.05);
        });

        // 下一行预览效果
        const showNextLine = pvConfig.showNextLine !== false;
        if (showNextLine) {
            for (let i = 0; i < wordsGrouped.length - 1; i++) {
                const currentLineEnd = lineTimeRanges[i].end;
                const nextLineDiv = lineDivs[i + 1];

                // 在当前行结束后显示下一行预览
                tl.set(nextLineDiv, { visibility: 'visible', opacity: 0.4, scale: 0.85 }, currentLineEnd - 0.2);
                tl.to(nextLineDiv, { opacity: 0.55, duration: 0.2 }, currentLineEnd - 0.2);

                // 下一句真正开始时恢复正常样式
                const nextLineStart = lineTimeRanges[i + 1].start;
                tl.to(nextLineDiv, { opacity: 1, scale: 1, duration: 0.15, clearProps: 'all' }, nextLineStart);
            }
        }

        // 逐字入场 + 卡拉OK填充动画
        wordsFlat.forEach((word, idx) => {
            const lineIdx = word.lineId;
            const lineDiv = lineDivs[lineIdx];
            if (!lineDiv) return;

            const charElements = lineDiv.querySelectorAll('.pv-char');
            const wordsInLine = wordsGrouped[lineIdx];
            const charIndexInLine = wordsInLine.findIndex(w => w.globalIndex === word.globalIndex);

            if (charIndexInLine === -1 || charIndexInLine >= charElements.length) return;

            const charEl = charElements[charIndexInLine];
            const fillEl = charEl.querySelector('.pv-char-fill');

            if (!charEl) return;

            // 入场动画
            const enterConfig = template.charEnter || {};
            const from = { ...enterConfig.from };
            const to = { ...enterConfig.to };
            const duration = enterConfig.duration || 0.12;
            const ease = enterConfig.ease || 'back.out(0.7)';

            tl.fromTo(charEl, from, { ...to, duration, ease }, word.start);

            // 卡拉OK填充动画
            if (fillEl && template.fillEffect !== 'none') {
                const fillDuration = word.end - word.start;
                const fillTiming = template.fillTiming || 'linear';

                tl.fromTo(fillEl,
                    { scaleX: 0 },
                    { scaleX: 1, duration: fillDuration, ease: fillTiming },
                    word.start
                );
            }

            // 退场动画
            const exitConfig = template.charExit || {};
            if (exitConfig.to) {
                const exitDuration = exitConfig.duration || 0.08;
                const exitEase = exitConfig.ease || 'power2.in';
                tl.to(charEl, { ...exitConfig.to, duration: exitDuration, ease: exitEase }, word.end - 0.05);
            }

            // 长音字震动效果
            if (word.isHeavy && template.shakeOnHeavy) {
                const shakeIntensity = template.shakeIntensity || { x: 3, y: 3, duration: 0.05, repeats: 2 };
                tl.to(charEl, {
                    x: `random(-${shakeIntensity.x}, ${shakeIntensity.x})`,
                    y: `random(-${shakeIntensity.y}, ${shakeIntensity.y})`,
                    duration: shakeIntensity.duration,
                    repeat: shakeIntensity.repeats,
                    yoyo: true,
                    ease: 'none'
                }, word.start);
            }
        });

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

            const { flat, grouped } = this.convertLrcData(motion.lrcData);

            if (grouped.length === 0) return;

            const template = this.getCurrentTemplate();
            const lineDivs = this.buildDOM(grouped);
            this.mainTimeline = this.buildTimeline(flat, grouped, lineDivs, template);

            // 同步到当前位置
            if (this.mainTimeline && position > 0) {
                const targetTime = position / 1000;

                // 找到当前行
                let currentLineStart = 0;
                for (let i = 0; i < grouped.length; i++) {
                    const lineWords = grouped[i];
                    if (lineWords[0] && targetTime >= lineWords[0].start) {
                        currentLineStart = lineWords[0].start;
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