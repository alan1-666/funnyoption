"use client";

import { useCallback, useEffect, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { getDepositAddress, requestWithdraw, type CustodyDepositAddress } from "@/lib/api";
import { formatAssetAmount, shortenAddress } from "@/lib/format";
import styles from "@/components/vault-console.module.css";

const ACCOUNTING_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS ?? "2");
const ACCOUNTING_STEP = ACCOUNTING_DECIMALS <= 0 ? "1" : `0.${"0".repeat(ACCOUNTING_DECIMALS - 1)}1`;
const COLLATERAL_SYMBOL = (process.env.NEXT_PUBLIC_COLLATERAL_SYMBOL ?? "USDT").toUpperCase();

type MainTab = "deposit" | "withdraw";

function validateWithdrawInput(rawAmount: string) {
  const trimmed = rawAmount.trim();
  if (!trimmed) throw new Error(`请输入提现金额（${COLLATERAL_SYMBOL}）。`);
  if (!/^\d+(\.\d+)?$/.test(trimmed)) throw new Error(`金额格式不正确。`);

  const [, decimals = ""] = trimmed.split(".");
  if (decimals.length > ACCOUNTING_DECIMALS) {
    throw new Error(`${COLLATERAL_SYMBOL} 最多支持 ${ACCOUNTING_DECIMALS} 位小数。`);
  }

  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || parsed <= 0) throw new Error(`金额必须大于 0。`);

  return Math.round(parsed * 10 ** ACCOUNTING_DECIMALS);
}

export function VaultConsole() {
  const { session, busy, prepareTrading } = useTradingSession();
  const [mainTab, setMainTab] = useState<MainTab>("deposit");
  const [depositAddr, setDepositAddr] = useState<CustodyDepositAddress | null>(null);
  const [addrLoading, setAddrLoading] = useState(false);
  const [addrError, setAddrError] = useState("");
  const [copied, setCopied] = useState(false);

  const [withdrawAmount, setWithdrawAmount] = useState("");
  const [recipient, setRecipient] = useState("");
  const [withdrawStatus, setWithdrawStatus] = useState("");
  const [withdrawBusy, setWithdrawBusy] = useState(false);

  const fetchAddress = useCallback(async () => {
    if (!session?.sessionId) return;
    setAddrLoading(true);
    setAddrError("");
    try {
      const addr = await getDepositAddress();
      setDepositAddr(addr);
    } catch (err) {
      setAddrError(err instanceof Error ? err.message : "获取充值地址失败");
    } finally {
      setAddrLoading(false);
    }
  }, [session?.sessionId]);

  useEffect(() => {
    if (session?.sessionId && mainTab === "deposit") {
      fetchAddress();
    }
  }, [session?.sessionId, mainTab, fetchAddress]);

  async function handleCopy() {
    if (!depositAddr?.address) return;
    try {
      await navigator.clipboard.writeText(depositAddr.address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  async function handleWithdraw() {
    setWithdrawStatus("");
    setWithdrawBusy(true);
    try {
      const amountInt = validateWithdrawInput(withdrawAmount);
      if (!recipient.trim()) throw new Error("请输入收款地址。");

      const result = await requestWithdraw(recipient.trim(), amountInt);
      setWithdrawStatus(`提现已提交（${result.withdraw_id.slice(0, 14)}…），状态：${result.status}`);
      setWithdrawAmount("");
    } catch (err) {
      setWithdrawStatus(err instanceof Error ? err.message : "提现失败");
    } finally {
      setWithdrawBusy(false);
    }
  }

  async function handleEnsureSession() {
    try {
      await prepareTrading();
    } catch {
      // Provider updates status
    }
  }

  const isLoggedIn = !!session?.sessionId;

  return (
    <section className={`panel ${styles.console}`}>
      <header className={styles.head}>
        <div>
          <span className="eyebrow">充值与提现</span>
          <p className={styles.sub}>
            连接钱包登录后，系统会分配专属充值地址；向该地址转入支持的币种即可自动折算为 {COLLATERAL_SYMBOL} 余额。
          </p>
        </div>
      </header>

      <div className={styles.tabs} role="tablist" aria-label="充值或提现">
        <button
          type="button"
          role="tab"
          aria-selected={mainTab === "deposit"}
          className={`${styles.tab} ${mainTab === "deposit" ? styles.tabActive : ""}`}
          onClick={() => setMainTab("deposit")}
        >
          充值
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={mainTab === "withdraw"}
          className={`${styles.tab} ${mainTab === "withdraw" ? styles.tabActive : ""}`}
          onClick={() => setMainTab("withdraw")}
        >
          提现
        </button>
      </div>

      {mainTab === "deposit" ? (
        <div className={styles.tabPanel} role="tabpanel">
          {!isLoggedIn ? (
            <>
              <p className={styles.tabHint}>请先连接钱包并授权交易密钥，系统将为你生成专属充值地址。</p>
              <button
                type="button"
                className={styles.primary}
                onClick={handleEnsureSession}
                disabled={busy !== null}
              >
                {busy === "connect" ? "连接中…" : busy === "session" ? "授权中…" : "连接钱包"}
              </button>
            </>
          ) : addrLoading ? (
            <p className={styles.tabHint}>正在获取充值地址…</p>
          ) : addrError ? (
            <>
              <p className={styles.tabHint} style={{ color: "var(--danger, #f87171)" }}>{addrError}</p>
              <button type="button" className={styles.primary} onClick={fetchAddress}>
                重试
              </button>
            </>
          ) : depositAddr ? (
            <>
              <p className={styles.tabHint}>
                向以下地址转入支持的币种（{depositAddr.chain} / {depositAddr.network}），到账后自动折算为 {COLLATERAL_SYMBOL} 余额。
              </p>

              <div className={styles.addressCard}>
                <span className={styles.addressLabel}>充值地址</span>
                <code className={styles.addressValue}>{depositAddr.address}</code>
                <button type="button" className={styles.copyBtn} onClick={handleCopy}>
                  {copied ? "已复制 ✓" : "复制地址"}
                </button>
              </div>

              <ul className={styles.infoList}>
                <li>链：{depositAddr.chain}</li>
                <li>网络：{depositAddr.network}</li>
                <li>支持币种：{(depositAddr.supported_coins ?? [depositAddr.coin]).join("、")}</li>
              </ul>

              <p className={styles.hint}>
                非 {COLLATERAL_SYMBOL} 币种（如 BNB）将按充值到账时的实时市场价格自动折算为 {COLLATERAL_SYMBOL} 入账。充值到账时间取决于链上确认速度。
              </p>
            </>
          ) : null}
        </div>
      ) : (
        <div className={styles.tabPanel} role="tabpanel">
          {!isLoggedIn ? (
            <>
              <p className={styles.tabHint}>请先连接钱包并授权交易密钥后发起提现。</p>
              <button
                type="button"
                className={styles.primary}
                onClick={handleEnsureSession}
                disabled={busy !== null}
              >
                {busy === "connect" ? "连接中…" : busy === "session" ? "授权中…" : "连接钱包"}
              </button>
            </>
          ) : (
            <>
              <p className={styles.tabHint}>输入金额和收款地址，提现将通过托管系统发送到链上。</p>

              <label className={styles.field}>
                <span className={styles.fieldLabel}>金额（{COLLATERAL_SYMBOL}）</span>
                <input
                  className={styles.inputWide}
                  value={withdrawAmount}
                  onChange={(e) => setWithdrawAmount(e.target.value)}
                  inputMode="decimal"
                  step={ACCOUNTING_STEP}
                  placeholder="0.00"
                />
              </label>

              <label className={styles.field}>
                <span className={styles.fieldLabel}>收款地址</span>
                <input
                  className={styles.inputWide}
                  value={recipient}
                  onChange={(e) => setRecipient(e.target.value)}
                  placeholder="0x…"
                />
              </label>

              <button
                type="button"
                className={styles.primary}
                onClick={handleWithdraw}
                disabled={withdrawBusy}
              >
                {withdrawBusy ? "提交中…" : "提现"}
              </button>
            </>
          )}

          {withdrawStatus ? <p className={styles.status}>{withdrawStatus}</p> : null}
        </div>
      )}
    </section>
  );
}
