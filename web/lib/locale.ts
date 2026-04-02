export function zhMarketStatus(status: string) {
  switch (String(status).toUpperCase()) {
    case "OPEN":
      return "交易中";
    case "RESOLVED":
      return "已结算";
    default:
      return status || "未知";
  }
}

export function zhGenericStatus(status: string) {
  switch (String(status).toUpperCase()) {
    case "NEW":
      return "挂单中";
    case "ACTIVE":
      return "生效中";
    case "COMPLETED":
      return "已完成";
    case "CREDITED":
      return "已入账";
    case "QUEUED":
      return "排队中";
    case "SUBMITTED":
      return "已提交";
    case "FAILED":
      return "失败";
    case "CANCELLED":
      return "已取消";
    case "FILLED":
      return "已成交";
    case "PARTIALLY_FILLED":
      return "部分成交";
    case "REVOKED":
      return "已撤销";
    case "PENDING":
      return "处理中";
    case "IN_USE":
      return "使用中";
    default:
      return status || "未知";
  }
}

export function zhOutcome(outcome: string) {
  switch (String(outcome).toUpperCase()) {
    case "YES":
      return "是";
    case "NO":
      return "否";
    default:
      return outcome || "未知";
  }
}

export function zhSide(side: string) {
  switch (String(side).toUpperCase()) {
    case "BUY":
      return "买入";
    case "SELL":
      return "卖出";
    default:
      return side || "未知";
  }
}
