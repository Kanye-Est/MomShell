#!/usr/bin/env npx tsx
/**
 * render-sand.ts — 用 Puppeteer 将沙滩 CSS 渲染为 sand.png
 *
 * 用法: npx tsx scripts/render-sand.ts [输出文件名]
 *
 * 可调参数在下方 CONFIG 区域，改完后重新运行即可。
 * 生成: src/assets/images/sand.png  (7680×630, RGBA, 透明顶部渐变到 #f8eed8)
 */

import puppeteer from 'puppeteer'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// ============ CONFIG ============
const WIDTH = 7680
const HEIGHT = 630
const OUTPUT = path.resolve(process.argv[2] || path.join(__dirname, '..', 'src', 'assets', 'images', 'sand.png'))
// ================================

const html = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
*, *::before, *::after { margin: 0; padding: 0; box-sizing: border-box; }
html, body { width: ${WIDTH}px; height: ${HEIGHT}px; overflow: hidden; background: transparent; }

.sand-layer {
  position: relative;
  width: ${WIDTH}px;
  height: ${HEIGHT}px;
}

.sand {
  width: 100%; height: 100%;
  background: linear-gradient(
    to bottom,
    rgba(100, 160, 200, 0) 0%,
    rgba(110, 165, 198, 0.08) 2%,
    rgba(130, 175, 192, 0.18) 4.5%,
    rgba(155, 188, 182, 0.32) 7.5%,
    rgba(178, 198, 168, 0.48) 11%,
    rgba(200, 202, 156, 0.62) 15%,
    rgba(218, 206, 155, 0.78) 19%,
    rgba(232, 210, 162, 0.92) 23%,
    #e0cca0 28%,
    #e6d4ae 38%,
    #ecdcba 52%,
    #f0e2c6 70%,
    #f4e8d0 86%,
    #f8eed8 100%
  );
}

.sand-wet {
  position: absolute;
  top: 0;
  width: 100%;
  height: 18%;
  background: linear-gradient(
    to bottom,
    rgba(80, 140, 180, 0.15),
    rgba(110, 158, 178, 0.06),
    transparent
  );
}

.tidal-zone {
  position: absolute;
  top: 22%;
  width: 100%;
  height: 74%;
  pointer-events: none;
  filter: url(#tidal-distort);
  background: linear-gradient(
    to top,
    rgba(145, 115, 78, 0.30) 0%,
    rgba(155, 125, 85, 0.24) 15%,
    rgba(168, 140, 98, 0.18) 30%,
    rgba(182, 158, 115, 0.10) 46%,
    rgba(198, 175, 135, 0.05) 60%,
    rgba(210, 190, 160, 0.02) 76%,
    rgba(220, 205, 175, 0.008) 88%,
    transparent 100%
  );
}

.tidal-zone-accent {
  position: absolute;
  top: 18%;
  width: 100%;
  height: 22%;
  pointer-events: none;
  filter: url(#tidal-distort-accent);
  background: linear-gradient(
    to top,
    transparent 0%,
    rgba(130, 100, 62, 0.06) 8%,
    rgba(135, 105, 68, 0.16) 22%,
    rgba(145, 115, 75, 0.28) 40%,
    rgba(162, 135, 95, 0.16) 62%,
    rgba(180, 155, 118, 0.05) 82%,
    transparent 100%
  );
}

.sand-grain {
  position: absolute;
  width: 2px; height: 2px;
  background: rgba(210, 195, 170, 0.5);
  border-radius: 50%;
}
</style>
</head>
<body>
<svg xmlns="http://www.w3.org/2000/svg" style="position:absolute;width:0;height:0;overflow:hidden;" aria-hidden="true">
  <defs>
    <filter id="tidal-distort" x="-15%" y="-15%" width="130%" height="130%" color-interpolation-filters="sRGB">
      <feTurbulence type="fractalNoise" baseFrequency="0.005 0.014" numOctaves="5" seed="37" result="noise"/>
      <feDisplacementMap in="SourceGraphic" in2="noise" scale="32" xChannelSelector="R" yChannelSelector="G" result="displaced"/>
      <feGaussianBlur in="displaced" stdDeviation="1.5" result="softened"/>
    </filter>
    <filter id="tidal-distort-accent" x="-15%" y="-15%" width="130%" height="130%" color-interpolation-filters="sRGB">
      <feTurbulence type="fractalNoise" baseFrequency="0.007 0.018" numOctaves="4" seed="91" result="noise"/>
      <feDisplacementMap in="SourceGraphic" in2="noise" scale="26" xChannelSelector="R" yChannelSelector="G" result="displaced"/>
      <feGaussianBlur in="displaced" stdDeviation="1.2" result="softened"/>
    </filter>
  </defs>
</svg>

<div class="sand-layer" id="sandLayer">
  <div class="sand"></div>
  <div class="sand-wet"></div>
  <div class="tidal-zone"></div>
  <div class="tidal-zone-accent"></div>
</div>

<script>
  // 固定种子的伪随机，保证每次渲染结果一致
  let seed = 42;
  function seededRandom() {
    seed = (seed * 16807 + 0) % 2147483647;
    return (seed - 1) / 2147483646;
  }

  const layer = document.getElementById('sandLayer');
  for (let i = 0; i < 80; i++) {
    const g = document.createElement('div');
    g.className = 'sand-grain';
    g.style.cssText =
      'left:' + (seededRandom() * 100) + '%;' +
      'top:' + (22 + seededRandom() * 78) + '%;' +
      'opacity:' + (0.2 + seededRandom() * 0.4) + ';' +
      'width:' + (1 + seededRandom() * 2) + 'px;' +
      'height:' + (1 + seededRandom() * 2) + 'px;';
    layer.appendChild(g);
  }
</script>
</body>
</html>`

async function main(): Promise<void> {
  const browser = await puppeteer.launch({
    headless: true,
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
    ],
  })

  try {
    const page = await browser.newPage()
    await page.setViewport({ width: WIDTH, height: HEIGHT, deviceScaleFactor: 1 })
    await page.setContent(html, { waitUntil: 'networkidle0' })

    // 等待 SVG filter 渲染完成
    await new Promise<void>(resolve => setTimeout(resolve, 500))

    await page.screenshot({
      path: OUTPUT,
      type: 'png',
      omitBackground: true,
      clip: { x: 0, y: 0, width: WIDTH, height: HEIGHT },
    })
  } finally {
    await browser.close()
  }
  console.log('sand.png saved to', OUTPUT, `(${WIDTH}x${HEIGHT})`)
}

main()
