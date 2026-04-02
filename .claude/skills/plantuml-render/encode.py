"""
将 PlantUML 代码编码为 plantuml.com 在线服务所需的 URL 参数。

用法：
    python3 encode.py <input.puml>
    cat input.puml | python3 encode.py

输出：编码后的字符串（无换行），可直接拼接到 URL：
    https://www.plantuml.com/plantuml/svg/<output>
"""

import zlib
import sys


def encode6bit(b):
    if b < 10:
        return chr(48 + b)
    b -= 10
    if b < 26:
        return chr(65 + b)
    b -= 26
    if b < 26:
        return chr(97 + b)
    b -= 26
    if b == 0:
        return '-'
    if b == 1:
        return '_'
    return '?'


def append3bytes(b1, b2, b3):
    c1 = b1 >> 2
    c2 = ((b1 & 0x3) << 4) | (b2 >> 4)
    c3 = ((b2 & 0xF) << 2) | (b3 >> 6)
    c4 = b3 & 0x3F
    return encode6bit(c1) + encode6bit(c2) + encode6bit(c3) + encode6bit(c4)


def encode(text):
    compressed = zlib.compress(text.encode('utf-8'), 9)[2:-4]
    result = ''
    i = 0
    while i < len(compressed):
        b1 = compressed[i] if i < len(compressed) else 0
        b2 = compressed[i + 1] if i + 1 < len(compressed) else 0
        b3 = compressed[i + 2] if i + 2 < len(compressed) else 0
        result += append3bytes(b1, b2, b3)
        i += 3
    return result


if __name__ == '__main__':
    if len(sys.argv) > 1:
        with open(sys.argv[1], 'r', encoding='utf-8') as f:
            text = f.read()
    else:
        text = sys.stdin.read()
    print(encode(text), end='')
