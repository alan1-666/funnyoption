# Frontend Surface Copy Guidelines

This file defines the default copy contract for user-facing product surfaces.

## Goal

Product UI copy should help users act, decide, and verify state.
It should not explain our design intent, implementation rationale, or internal
layout choices.

## Hard rules

1. Do not show meta/design-rationale copy in the product UI.
   - forbidden examples:
     - “这里会显示……”
     - “把……收成……”
     - “像 X 一样……”
     - “不再像以前那样……”
     - “这一块是为了……”

2. Prefer factual, product-facing labels.
   - good:
     - `我的订单`
     - `暂无活动挂单`
     - `等待裁决`
   - bad:
     - `把下单动作收紧到右侧一条 rail`

3. Empty states must be short and literal.
   - default shape:
     - one short sentence
     - state first, no internal explanation
   - examples:
     - `暂无活动挂单`
     - `暂无历史结果`
     - `连接钱包后查看订单`

4. Headlines should name the user task or state, not the implementation.
   - good:
     - `交易面板`
     - `市场详情`
     - `市场时间线`
   - bad:
     - `像 Worm 一样把事件、赔率和时间线收成一个主舞台`

5. Supporting copy is optional, not mandatory.
   - if a section already has a clear title, chart, table, or state, do not add
     filler text just to avoid blank space

## Preferred tone

- concise
- factual
- action-oriented
- calm

## Review checklist

Before shipping a frontend change, check:

- does any sentence describe the page design instead of the product state?
- does any sentence explain where content “will” appear instead of showing it?
- can any paragraph be deleted without losing user understanding?
- can the same idea be expressed as a shorter title, badge, or empty state?

If yes, shorten or delete it.
