# Markdown 内容编写与代码块折叠

QUTCraft Platform 的内容正文以标准 Markdown 保存并通过 Content API 原样传输。默认 MD3 门户、后台实时预览、智能体提案预览和活动方案预览使用同一套安全渲染器；渲染结果经过 HTML 清洗，正文中的脚本、事件处理器和不安全地址不会被执行。

## 代码块默认行为

- 所有围栏代码块都显示语言、行数和“复制”按钮；复制在 HTTPS、localhost 与普通 HTTP 部署中均提供兼容处理。
- 未配置时使用 `auto`：达到 12 行的代码块默认折叠，较短代码块直接展开。
- 行内代码（单个反引号）不参与折叠。
- 自定义门户可以忽略这些显示选项，但必须继续把 Markdown 当作不可信输入清洗。

## 文章级 Front Matter

Front Matter 必须位于正文第一行。`code-fold` 控制文章中的全部围栏代码块：

```markdown
---
code-fold: true
---

# 标题
```

| 值 | 行为 |
| --- | --- |
| `auto` | 默认规则：12 行及以上折叠，短代码直接展开。 |
| `true` | 所有代码块都可折叠，并在首次加载时收起。 |
| `show` | 所有代码块都可折叠，但在首次加载时展开。 |
| `false` | 所有代码块直接展开，不显示折叠控制。复制按钮仍保留。 |

为了兼容导入工具，也接受 `collapsed`、`expanded`、`disabled` 等同义值；新文章应优先使用表中的四个值。Front Matter 只影响呈现，不会被显示为正文，也不改变 API 字段或数据库结构。

## 单个代码块覆盖

单块设置优先于文章级 Front Matter。推荐在围栏信息后添加参数：

````markdown
```python {code-fold=true}
print("这个代码块默认收起")
```

```json {code-fold=show}
{"status": "这个代码块默认展开"}
```

```text {code-fold=false}
这个代码块不显示折叠控制
```
````

也可以使用便于非技术编辑者辨认的包裹语法：

````markdown
<code-fold: true>

```python
print("这个代码块默认收起")
```

</code-fold>
````

包裹标记必须单独占行，内部只能包裹一个完整围栏代码块，并使用 `</code-fold>` 结束。支持的值与 Front Matter 相同。标记不完整或值无效时，渲染器不会吞掉其中的文章内容。

## API 与自定义门户约定

服务端不解析或改写折叠设置，`body` 始终是 Markdown 字符串。默认门户负责移除 Front Matter、解释折叠选项、生成复制按钮并清洗最终 HTML。自定义门户实现者可以采用相同行为，也可以忽略展示选项并按普通围栏代码渲染，但不得直接把未经清洗的 HTML 注入页面。

