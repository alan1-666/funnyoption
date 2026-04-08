"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import {
  depositCollateralWithAutoApprove,
  depositNativeToVault,
  ensureTargetChain,
  getChainMeta,
  queueWithdrawal
} from "@/lib/chain";
import { shortenAddress } from "@/lib/format";
import styles from "@/components/vault-console.module.css";

const ACCOUNTING_DECIMALS = Number(process.env.NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS ?? "2");
const ACCOUNTING_STEP = ACCOUNTING_DECIMALS <= 0 ? "1" : `0.${"0".repeat(ACCOUNTING_DECIMALS - 1)}1`;

function validateAmountInput(rawAmount: string, tokenSymbol: string) {
  const trimmed = rawAmount.trim();
  if (!trimmed) {
    throw new Error(`请输入金额（${tokenSymbol}）。`);
  }
  if (!/^\d+(\.\d+)?$/.test(trimmed)) {
    throw new Error(`金额格式不正确（${tokenSymbol}）。`);
  }

  const [, decimals = ""] = trimmed.split(".");
  if (decimals.length > ACCOUNTING_DECIMALS) {
    throw new Error(`${tokenSymbol} 最多支持 ${ACCOUNTING_DECIMALS} 位小数。`);
  }

  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(`金额（${tokenSymbol}）必须大于 0。`);
  }
  return trimmed;
}

function explorerTxUrl(explorerBase: string, txHash: string) {
  const base = explorerBase.replace(/\/$/, "");
  return `${base}/tx/${txHash}`;
}

export function VaultConsole() {
  const { wallet, busy, connect } = useTradingSession();
  const chain = useMemo(() => getChainMeta(), []);
  const [depositAmount, setDepositAmount] = useState("100");
  const [nativeDepositAmount, setNativeDepositAmount] = useState("0.1");
  const [withdrawAmount, setWithdrawAmount] = useState("50");
  const [recipient, setRecipient] = useState("");
  const [status, setStatus] = useState("");
  const [recentTx, setRecentTx] = useState<{ label: string; hash: string }[]>([]);

  function rememberTx(label: string, hash: string) {
    setRecentTx((prev) => [...prev, { label, hash }].slice(-6));
  }

  async function ensureWallet() {
    if (wallet?.walletAddress) {
      return wallet.walletAddress;
    }
    const conn = await connect();
    const addr = conn?.walletAddress;
    if (!addr) {
      throw new Error("请先连接钱包。");
    }
    return addr;
  }

  async function handleCollateralDeposit() {
    setStatus("");
    try {
      const addr = await ensureWallet();
      const normalizedAmount = validateAmountInput(depositAmount, chain.tokenSymbol);
      setStatus("正在发起充值（首次会请求对 Vault 的无限额授权，随后再签充值）…");
      const { approveTxHash, depositTxHash } = await depositCollateralWithAutoApprove(addr, normalizedAmount);
      if (approveTxHash) {
        rememberTx("授权", approveTxHash);
        setStatus(`授权已提交。`);
      }
      rememberTx("充值", depositTxHash);
      setStatus(approveTxHash ? "充值已提交（本次已提交无限额授权交易）。" : "充值已提交。");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "充值失败");
    }
  }

  async function handleNativeDeposit() {
    setStatus("");
    try {
      const addr = await ensureWallet();
      const trimmed = nativeDepositAmount.trim();
      if (!trimmed || !/^\d+(\.\d+)?$/.test(trimmed) || Number(trimmed) <= 0) {
        throw new Error(`请输入有效的 ${chain.nativeCurrencySymbol} 金额。`);
      }
      setStatus(`${chain.nativeCurrencySymbol} 充值提交中…`);
      await ensureTargetChain();
      const txHash = await depositNativeToVault(addr, trimmed);
      rememberTx(`${chain.nativeCurrencySymbol} 充值`, txHash);
      setStatus("已提交。");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : `${chain.nativeCurrencySymbol} 充值失败`);
    }
  }

  async function handleWithdrawal() {
    setStatus("");
    try {
      const addr = await ensureWallet();
      if (!recipient.trim()) {
        throw new Error("请输入收款地址。");
      }
      const normalizedAmount = validateAmountInput(withdrawAmount, chain.tokenSymbol);
      setStatus("提现请求提交中…");
      await ensureTargetChain();
      const { txHash, withdrawalId } = await queueWithdrawal(addr, normalizedAmount, recipient.trim());
      rememberTx("提现", txHash);
      setStatus(`提现已提交。请求 ID：${withdrawalId.slice(0, 10)}…`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "提现失败");
    }
  }

  const explorer = chain.explorerUrl?.trim() ?? "";
  const walletLabel = wallet ? shortenAddress(wallet.walletAddress) : null;

  return (
    <section className={`panel ${styles.console}`}>
      <header className={styles.head}>
        <div>
          <span className="eyebrow">充值与提现</span>
          <p className={styles.sub}>
            {chain.tokenSymbol}：若从未授权过 Vault，会先无限额授权（签一次）再充值（再签一次）；同一钱包之后每次充值通常只需签充值一笔。{chain.nativeCurrencySymbol} 无需授权，一笔即可。
          </p>
        </div>
        {walletLabel ? <span className="pill">{walletLabel}</span> : null}
      </header>

      <div className={styles.block}>
        <h2 className={styles.blockTitle}>充值</h2>

        <div className={styles.field}>
          <span className={styles.fieldLabel}>{chain.tokenSymbol}</span>
          <div className={styles.fieldRow}>
            <input
              className={styles.input}
              value={depositAmount}
              onChange={(e) => setDepositAmount(e.target.value)}
              inputMode="decimal"
              step={ACCOUNTING_STEP}
              aria-label={`${chain.tokenSymbol} 金额`}
            />
            <button type="button" className={styles.primary} onClick={handleCollateralDeposit} disabled={busy === "connect"}>
              {busy === "connect" ? "连接中…" : `充值 ${chain.tokenSymbol}`}
            </button>
          </div>
        </div>

        <div className={styles.divider} />

        <div className={styles.field}>
          <span className={styles.fieldLabel}>{chain.nativeCurrencySymbol}</span>
          <div className={styles.fieldRow}>
            <input
              className={styles.input}
              value={nativeDepositAmount}
              onChange={(e) => setNativeDepositAmount(e.target.value)}
              inputMode="decimal"
              step="0.001"
              aria-label={`${chain.nativeCurrencySymbol} 金额`}
            />
            <button type="button" className={styles.secondary} onClick={handleNativeDeposit} disabled={busy === "connect"}>
              充值 {chain.nativeCurrencySymbol}
            </button>
          </div>
        </div>
      </div>

      <div className={styles.block}>
        <h2 className={styles.blockTitle}>提现</h2>
        <div className={styles.field}>
          <span className={styles.fieldLabel}>金额（{chain.tokenSymbol}）</span>
          <input
            className={styles.inputWide}
            value={withdrawAmount}
            onChange={(e) => setWithdrawAmount(e.target.value)}
            inputMode="decimal"
            step={ACCOUNTING_STEP}
          />
        </div>
        <div className={styles.field}>
          <span className={styles.fieldLabel}>收款地址</span>
          <input
            className={styles.inputWide}
            value={recipient}
            onChange={(e) => setRecipient(e.target.value)}
            placeholder={wallet?.walletAddress ?? "0x…"}
          />
        </div>
        <button type="button" className={styles.primary} onClick={handleWithdrawal} disabled={busy === "connect"}>
          提现
        </button>
      </div>

      {!wallet ? (
        <p className={styles.hint}>
          尚未连接钱包。点击上方「充值」或「提现」会先弹出连接；也可用交易页的连接入口。
        </p>
      ) : null}

      {recentTx.length > 0 ? (
        <ul className={styles.txList}>
          {recentTx.map((row) => (
            <li key={`${row.label}-${row.hash}`}>
              <span className={styles.txLabel}>{row.label}</span>
              {explorer ? (
                <a className={styles.txLink} href={explorerTxUrl(explorer, row.hash)} target="_blank" rel="noreferrer">
                  {row.hash}
                </a>
              ) : (
                <code className={styles.txHash}>{row.hash}</code>
              )}
            </li>
          ))}
        </ul>
      ) : null}

      {status ? <p className={styles.status}>{status}</p> : null}
    </section>
  );
}
