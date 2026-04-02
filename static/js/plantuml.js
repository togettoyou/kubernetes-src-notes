(function () {
  function encode6bit(b) {
    if (b < 10) return String.fromCharCode(48 + b);
    b -= 10;
    if (b < 26) return String.fromCharCode(65 + b);
    b -= 26;
    if (b < 26) return String.fromCharCode(97 + b);
    b -= 26;
    if (b === 0) return '-';
    if (b === 1) return '_';
    return '?';
  }

  function encodeBytes(bytes) {
    let result = '';
    for (let i = 0; i < bytes.length; i += 3) {
      const b1 = bytes[i] || 0;
      const b2 = bytes[i + 1] || 0;
      const b3 = bytes[i + 2] || 0;
      const c1 = b1 >> 2;
      const c2 = ((b1 & 0x3) << 4) | (b2 >> 4);
      const c3 = ((b2 & 0xf) << 2) | (b3 >> 6);
      const c4 = b3 & 0x3f;
      result += encode6bit(c1) + encode6bit(c2) + encode6bit(c3) + encode6bit(c4);
    }
    return result;
  }

  async function encode(text) {
    const input = new TextEncoder().encode(text);
    const cs = new CompressionStream('deflate-raw');
    const writer = cs.writable.getWriter();
    writer.write(input);
    writer.close();
    const chunks = [];
    const reader = cs.readable.getReader();
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      chunks.push(value);
    }
    const totalLength = chunks.reduce((sum, c) => sum + c.length, 0);
    const compressed = new Uint8Array(totalLength);
    let offset = 0;
    for (const chunk of chunks) {
      compressed.set(chunk, offset);
      offset += chunk.length;
    }
    return encodeBytes(compressed);
  }

  async function render() {
    const blocks = Array.from(document.querySelectorAll('pre code')).filter(
      (el) => el.textContent.trimStart().startsWith('@startuml')
    );
    for (const block of blocks) {
      const pre = block.closest('pre');
      try {
        const encoded = await encode(block.textContent);
        const img = document.createElement('img');
        img.src = 'https://www.plantuml.com/plantuml/svg/' + encoded;
        img.alt = 'PlantUML Diagram';
        img.style.maxWidth = '100%';
        pre.replaceWith(img);
      } catch (e) {
        console.error('PlantUML render error:', e);
      }
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', render);
  } else {
    render();
  }
})();
