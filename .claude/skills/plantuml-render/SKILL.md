---
name: plantuml-render
description: PlantUML 渲染助手。扫描指定文章中的所有 PlantUML 代码块，通过 plantuml.com 在线服务生成 SVG 矢量图，保存到 static 目录下，在 details 外插入图片引用，代码块本身保留不动。当用户想渲染文章中的 PlantUML 图表时触发。
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash, Edit, Write
---

# PlantUML 渲染助手

通过 plantuml.com 在线服务将文章中的 PlantUML 代码块渲染为 SVG 矢量图，存入 static 目录，在代码块下方插入图片引用，代码块本身保留不动。

## 调用方式

用户调用格式：

    /plantuml-render <文章文件路径>

示例：

    /plantuml-render content/extensions/_index.md
    /plantuml-render content/kube-scheduler/05.md

参数说明：
- `$1`（文章文件路径）：相对于项目根目录的文章路径

用户原始输入：`$ARGUMENTS`

---

## 执行步骤

### 第一步：解析参数

从 `$ARGUMENTS` 中提取文章文件路径。如果未提供路径，直接询问用户，不要猜测。

### 第二步：确定 static 输出目录

根据文章路径推导 static 输出目录，规则如下：

- 去掉 `content/` 前缀
- 如果是 `_index.md`，去掉 `/_index.md`，剩余部分作为目录名
- 如果是普通文章如 `05.md`，去掉 `.md` 后缀，剩余部分作为目录名

示例：
- `content/extensions/_index.md` → `static/extensions/`
- `content/kube-scheduler/05.md` → `static/kube-scheduler/05/`
- `content/extensions/api-extension.md` → `static/extensions/api-extension/`

使用 Bash 创建该目录（如不存在）。

### 第三步：检测依赖

检测以下工具是否可用：

1. `python --version`（Windows）或 `python3 --version`（macOS/Linux）：用于 PlantUML 编码
2. `curl --version`：用于请求在线服务

若两者均不可用，告知用户安装后重试，停止执行。

### 第四步：读取文章，提取 PlantUML 代码块

读取文章文件，找出所有 plantuml 代码块，对每个代码块记录：

- 完整原始文本（含起始行和结尾行），用于后续精确定位
- 代码块是否已被 `<details><summary>展开/收起</summary>` 包裹（向上查找最近的 `<details>` 标签判断）
- 代码块序号（从 1 开始），用于生成图片文件名

若文章中没有 PlantUML 代码块，告知用户并停止执行。

### 第五步：逐个渲染 SVG

对每个 PlantUML 代码块：

**1. 提取图片名**

- 若 `@startuml` 后跟有名称（如 `@startuml layered-arch`），使用该名称
- 否则使用 `diagram-01`、`diagram-02` 依序命名

**2. 写入临时文件**

将代码块内容写入 `/tmp/plantuml-<name>.puml`（代码块原样写入，不做任何修改）。

**3. 编码并请求 SVG**

使用 skill 目录下的 `encode.py` 将 PlantUML 代码编码为 URL 参数，再通过 curl 获取 SVG：

    # Windows 用 python，macOS/Linux 用 python3
    ENCODED=$(python .claude/skills/plantuml-render/encode.py "$TMPFILE")
    curl -sL "https://www.plantuml.com/plantuml/svg/${ENCODED}" -o "${TMPFILE%.puml}.svg"

**4. 移动 SVG 到 static 目录**

将 `/tmp/plantuml-<name>.svg` 移动到第二步确定的 static 输出目录，文件名为 `<name>.svg`。

**5. 清理临时文件**

### 第六步：更新文章

代码块内容**不做任何修改**。

对每个成功渲染的代码块，按以下两种情况处理：

**情况一：代码块已在 `<details>` 内**

在 `</details>` 标签之后换行插入图片引用：

    <details><summary>展开/收起</summary>

    ```plantuml
    （原始代码块保持不变）
    ```

    </details>

    ![](/extensions/diagram-01.svg)

**情况二：代码块未被 `<details>` 包裹**

用 `<details>` 将代码块包裹，再在 `</details>` 后插入图片引用：

    <details><summary>展开/收起</summary>

    ```plantuml
    （原始代码块保持不变）
    ```

    </details>

    ![](/extensions/diagram-01.svg)

注意：
- 图片路径以 `/` 开头，相对于 static 根目录（Hugo 规范）
- 使用 Edit 工具精确定位并修改，不要大段替换

### 第七步：汇报结果

输出简短摘要：

    渲染完成：
      ✓ diagram-01.svg → static/extensions/diagram-01.svg
      （共 1 张图片）

    文章已更新：content/extensions/_index.md

若有渲染失败的图表，报告错误信息，保留原始代码块不做任何修改。
