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
                lineEnter: {
                    from: { opacity: 0 },
                    to: { opacity: 1 },
                    duration: 0.12,
                    ease: 'power2.out'
                },
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
                lineExit: {
                    duration: 0.1,
                    ease: 'power1.in',
                    hideDelay: 0.05
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
                lineEnter: {
                    from: { opacity: 0 },
                    to: { opacity: 1 },
                    duration: 0.15,
                    ease: 'power2.out'
                },
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
                lineExit: {
                    duration: 0.1,
                    ease: 'power2.in',
                    hideDelay: 0.05
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
                lineEnter: {
                    from: { opacity: 0, y: -10 },
                    to: { opacity: 1, y: 0 },
                    duration: 0.15,
                    ease: 'power2.out'
                },
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
                lineExit: {
                    duration: 0.1,
                    ease: 'power1.in',
                    hideDelay: 0.05
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

    // 构建单行 DOM
    // line: 单行歌词数据，lineIdx: 该行在整个歌词中的索引
    buildLineDOM(line, lineIdx) {
        const lineDiv = document.createElement('div');
        lineDiv.className = 'pv-line';
        lineDiv.setAttribute('data-line-idx', lineIdx);

        const words = line.words || [];
        if (words.length > 0) {
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

        lineDiv.style.opacity = '0';
        lineDiv.style.visibility = 'hidden';
        return lineDiv;
    }

    // 构建 DOM 结构，只构建当前行和下一行（预览用）。
    // lrcData: 全部歌词数据，currentIndex: 当前行索引
    // 返回 { [lineIdx]: lineDivElement } 格式的映射表
    buildDOM(lrcData, currentIndex) {
        if (!this.container || !lrcData || !lrcData.length) return {};

        this.container.innerHTML = '';
        const lineDivs = {};
        const pvConfig = this.getPVConfig();
        const showNext = pvConfig.showNextLine !== false;

        const currLine = lrcData[currentIndex];
        if (currLine) {
            const div = this.buildLineDOM(currLine, currentIndex);
            this.container.appendChild(div);
            lineDivs[currentIndex] = div;
        }

        if (showNext && currentIndex + 1 < lrcData.length) {
            const nextLine = lrcData[currentIndex + 1];
            const div = this.buildLineDOM(nextLine, currentIndex + 1);
            this.container.appendChild(div);
            lineDivs[currentIndex + 1] = div;
        }

        return lineDivs;
    }

    // 构建 GSAP Timeline
    // lrcData: 原始歌词数据，lineDivMap: { [lineIdx]: lineDivElement }，仅包含当前行和下一行
    buildTimeline(lrcData, lineDivMap, template) {
        const pvConfig = this.getPVConfig();
        const tl = gsap.timeline({ paused: true });
        const animLog = [];

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

        const lineIndices = Object.keys(lineDivMap).map(Number).sort((a, b) => a - b);
        if (!lineIndices.length) return tl;

        // 计算已构建行的时间范围
        const lineTimeRanges = {};
        for (const idx of lineIndices) {
            const line = lrcData[idx];
            if (line) {
                lineTimeRanges[idx] = {
                    start: line.time / 1000,
                    end: (line.endTime || line.time + 2000) / 1000
                };
            }
        }

        // 整行显隐动画
        for (const lineIdx of lineIndices) {
            const lineStart = lineTimeRanges[lineIdx].start;
            const lineEnd = lineTimeRanges[lineIdx].end;
            const lineDiv = lineDivMap[lineIdx];
            if (!lineDiv) continue;

            const lineEnter = template.lineEnter || {
                from: { opacity: 0 },
                to: { opacity: 1 },
                duration: 0.12,
                ease: 'power2.out'
            };

            tl.set(lineDiv, { visibility: 'visible', ...lineEnter.from }, lineStart);
            logAnim(lineIdx, '整行显示', lineStart, 0, `行 ${lineIdx}`);

            tl.to(lineDiv, { ...lineEnter.to, duration: lineEnter.duration, ease: lineEnter.ease }, lineStart);
            logAnim(lineIdx, '整行淡入', lineStart, lineEnter.duration, `行 ${lineIdx}`);

            const lineExit = template.lineExit || {
                duration: 0.1,
                ease: 'power1.in',
                hideDelay: 0.05
            };

            tl.to(lineDiv, { opacity: 0, duration: lineExit.duration, ease: lineExit.ease }, lineEnd);
            logAnim(lineIdx, '整行淡出', lineEnd, lineExit.duration, `行 ${lineIdx}`);

            tl.set(lineDiv, { visibility: 'hidden' }, lineEnd + (lineExit.hideDelay || 0.05));
            logAnim(lineIdx, '整行隐藏', lineEnd + (lineExit.hideDelay || 0.05), 0, `行 ${lineIdx}`);
        }

        // 下一行预览：当前行结束时预览下一行
        const showNextLine = pvConfig.showNextLine !== false;
        if (showNextLine) {
            for (const lineIdx of lineIndices) {
                const nextIdx = lineIdx + 1;
                if (!lineDivMap[nextIdx]) continue;

                const currentLineEnd = lineTimeRanges[lineIdx].end;
                const nextLineStart = lineTimeRanges[nextIdx].start;
                const nextLineDiv = lineDivMap[nextIdx];

                tl.set(nextLineDiv, { visibility: 'visible', opacity: 0.4, scale: 0.85 }, currentLineEnd - 0.2);
                logAnim(nextIdx, '下一行预览显示', currentLineEnd - 0.2, 0, `行 ${nextIdx}`);

                tl.to(nextLineDiv, { opacity: 0.55, duration: 0.2 }, currentLineEnd - 0.2);
                logAnim(nextIdx, '下一行预览淡入', currentLineEnd - 0.2, 0.2, `行 ${nextIdx}`);

                tl.to(nextLineDiv, { opacity: 1, scale: 1, duration: 0.15 }, nextLineStart);
                logAnim(nextIdx, '下一行恢复样式', nextLineStart, 0.15, `行 ${nextIdx}`);
            }
        }

        // 逐字入场 + 填充动画
        for (const lineIdx of lineIndices) {
            const lineDiv = lineDivMap[lineIdx];
            const line = lrcData[lineIdx];
            if (!lineDiv || !line) continue;

            const charElements = lineDiv.querySelectorAll('.pv-char');
            const words = line.words || [];

            if (words.length > 0) {
                words.forEach((word, wordIdx) => {
                    const wordStart = word.time / 1000;
                    const wordEnd = wordIdx < words.length - 1
                        ? words[wordIdx + 1].time / 1000
                        : (line.endTime || line.time + 2000) / 1000;

                    if (wordIdx >= charElements.length) return;

                    const charEl = charElements[wordIdx];
                    const fillEl = charEl.querySelector('.pv-char-fill');

                    const enterConfig = template.charEnter || {};
                    const from = { ...enterConfig.from };
                    const to = { ...enterConfig.to };
                    const duration = enterConfig.duration || 0.12;
                    const ease = enterConfig.ease || 'back.out(0.7)';

                    tl.fromTo(charEl, from, {
                        ...to,
                        duration,
                        ease,
                        data: { key: `${lineIdx}_${wordIdx}`, text: word.text },
                        onStart: function () {
                            if (window.__pvDebug) {
                                const p = this.parent;
                                console.log(
                                    `[DBG:fromTo] L${this.data.key} "${this.data.text}"`,
                                    `tl=${p ? p.time().toFixed(3) : '?'}`,
                                    `ratio=${this.ratio?.toFixed(4) ?? '?'}`,
                                    `paused=${p ? p.paused() : '?'}`
                                );
                            }
                        }
                    }, wordStart);
                    logAnim(lineIdx, '字符入场', wordStart, duration, `"${word.text}"`);

                    if (fillEl && template.fillEffect !== 'none') {
                        tl.set(fillEl, { scaleX: 1 }, wordStart);
                        if (window.__pvDebug) {
                            console.log(`[DBG:fill] L${lineIdx}_${wordIdx} set scaleX:1 at ${wordStart.toFixed(3)}`);
                        }
                        logAnim(lineIdx, '填充', wordStart, 0, `"${word.text}"填充层`);
                    }

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
                const text = line.text || '';
                const chars = text.split('').filter(c => c.trim());
                const lineStart = line.time / 1000;
                const lineEnd = (line.endTime || line.time + 2000) / 1000;

                chars.forEach((char, charIdx) => {
                    const charStart = lineStart + (charIdx / Math.max(chars.length, 1)) * (lineEnd - lineStart);
                    const charEnd = charIdx < chars.length - 1
                        ? lineStart + ((charIdx + 1) / Math.max(chars.length, 1)) * (lineEnd - lineStart)
                        : lineEnd;

                    if (charIdx >= charElements.length) return;

                    const charEl = charElements[charIdx];
                    const fillEl = charEl.querySelector('.pv-char-fill');

                    const enterConfig = template.charEnter || {};
                    const from = { ...enterConfig.from };
                    const to = { ...enterConfig.to };
                    const duration = enterConfig.duration || 0.12;
                    const ease = enterConfig.ease || 'back.out(0.7)';

                    tl.fromTo(charEl, from, {
                        ...to,
                        duration,
                        ease,
                        data: { key: `${lineIdx}_${charIdx}`, text: char },
                        onStart: function () {
                            if (window.__pvDebug) {
                                const p = this.parent;
                                console.log(
                                    `[DBG:fromTo] L${this.data.key} "${this.data.text}"`,
                                    `tl=${p ? p.time().toFixed(3) : '?'}`,
                                    `ratio=${this.ratio?.toFixed(4) ?? '?'}`,
                                    `paused=${p ? p.paused() : '?'}`
                                );
                            }
                        }
                    }, charStart);
                    logAnim(lineIdx, '字符入场', charStart, duration, `"${char}"`);

                    if (fillEl && template.fillEffect !== 'none') {
                        tl.set(fillEl, { scaleX: 1 }, charStart);
                        if (window.__pvDebug) {
                            console.log(`[DBG:fill] L${lineIdx}_${charIdx} set scaleX:1 at ${charStart.toFixed(3)}`);
                        }
                        logAnim(lineIdx, '填充', charStart, 0, `"${char}"填充层`);
                    }
                });
            }
        }

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

        // 换行检测：只构建当前行和下一行的 DOM + Timeline
        if (this.lastLineIndex !== currentIndex) {
            this._rebuildCount = (this._rebuildCount || 0) + 1;
            if (window.__pvDebug) {
                console.log(
                    `[DBG:rebuild] #${this._rebuildCount} idx=${currentIndex}`,
                    `pos=${position}`, `lastLine=${this.lastLineIndex}`
                );
            }
            this.lastLineIndex = currentIndex;
            this.reset();

            if (motion.lrcData.length === 0) return;

            const template = this.getCurrentTemplate();
            const lineDivMap = this.buildDOM(motion.lrcData, currentIndex);
            this.mainTimeline = this.buildTimeline(motion.lrcData, lineDivMap, template);

            // 同步到当前位置
            if (this.mainTimeline && position > 0) {
                const targetTime = position / 1000;
                const tlDuration = this.mainTimeline.duration();

                if (targetTime > 0 && targetTime < tlDuration) {
                    if (window.__pvDebug) {
                        console.log(
                            `[DBG:seek] target=${targetTime.toFixed(3)} tlDur=${tlDuration.toFixed(3)}`
                        );
                    }
                    this.mainTimeline.seek(targetTime);
                } else if (window.__pvDebug) {
                    console.log(`[DBG:seek] SKIP target=${targetTime.toFixed(3)} tlDur=${tlDuration.toFixed(3)}`);
                }

                if (window.__pvDebug) {
                    console.log(`[DBG:state] isPlaying=${isPlaying}`);
                }
                if (isPlaying) {
                    this.mainTimeline.play();
                } else {
                    this.mainTimeline.pause();
                }
            }
        }

        // 处理播放状态（缓存 seek 位置避免空转）
        if (this.mainTimeline) {
            const currentPos = frameData.position / 1000;
            const duration = this.mainTimeline.duration();

            if (!isPlaying) {
                // 仅当位置变化超过阈值时才 seek
                if (Math.abs(currentPos - (this._lastSeekPos || 0)) > 0.005) {
                    this._lastSeekPos = currentPos;
                    if (currentPos >= 0 && currentPos <= duration) {
                        if (window.__pvDebug) {
                            console.log(`[DBG:pause-seek] pos=${currentPos.toFixed(3)} tlDur=${duration.toFixed(3)}`);
                        }
                        this.mainTimeline.seek(currentPos);
                    }
                }
                // 确保暂停状态
                if (!this.mainTimeline.paused()) {
                    if (window.__pvDebug) {
                        console.log(`[DBG:pause] force pause at tl=${this.mainTimeline.time().toFixed(3)}`);
                    }
                    this.mainTimeline.pause();
                }
            } else {
                this._lastSeekPos = currentPos;
                // GSAP 内部 ticker 驱动动画，仅在 timeline 非预期停止时恢复
                if (this.mainTimeline.paused() && currentPos < duration - 0.01) {
                    if (window.__pvDebug) {
                        console.log(
                            `[DBG:play] resume tl=${this.mainTimeline.time().toFixed(3)}`,
                            `cur=${currentPos.toFixed(3)} dur=${duration.toFixed(3)}`
                        );
                    }
                    this.mainTimeline.play();
                } else if (window.__pvDebug && this.mainTimeline.paused()) {
                    console.log(
                        `[DBG:play] SKIP tl=${this.mainTimeline.time().toFixed(3)}`,
                        `cur=${currentPos.toFixed(3)} dur=${duration.toFixed(3)}`,
                        `reason=${currentPos >= duration - 0.01 ? 'atEnd' : 'notPaused'}`
                    );
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