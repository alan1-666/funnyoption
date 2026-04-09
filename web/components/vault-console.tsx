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

type MainTab = "deposit" | "withdraw";
type DepositAsset = "collateral" | "native";

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
  const [mainTab, setMainTab] = useState<MainTab>("deposit");
  const [depositAsset, setDepositAsset] = useState<DepositAsset>("collateral");
  const [collateralAmount, setCollateralAmount] = useState("100");
  const [nativeAmount, setNativeAmount] = useState("0.1");
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
    const addr = await ensureWallet();
    const normalizedAmount = validateAmountInput(collateralAmount, chain.tokenSymbol);
    setStatus("正在发起充值（首次会请求对 Vault 的无限额授权，随后再签充值）…");
    const { approveTxHash, depositTxHash } = await depositCollateralWithAutoApprove(addr, normalizedAmount);
    if (approveTxHash) {
      rememberTx("授权", approveTxHash);
      setStatus("授权已提交。");
    }
    rememberTx("充值", depositTxHash);
    setStatus(approveTxHash ? "充值已提交（本次已提交无限额授权交易）。" : "充值已提交。");
  }

  async function handleNativeDeposit() {
    const addr = await ensureWallet();
    const trimmed = nativeAmount.trim();
    if (!trimmed || !/^\d+(\.\d+)?$/.test(trimmed) || Number(trimmed) <= 0) {
      throw new Error(`请输入有效的 ${chain.nativeCurrencySymbol} 金额。`);
    }
    setStatus(`${chain.nativeCurrencySymbol} 充值提交中…`);
    await ensureTargetChain();
    const txHash = await depositNativeToVault(addr, trimmed);
    rememberTx(`${chain.nativeCurrencySymbol} 充值`, txHash);
    setStatus("已提交。");
  }

  async function handleDepositSubmit() {
    setStatus("");
    try {
      if (depositAsset === "collateral") {
        await handleCollateralDeposit();
      } else {
        await handleNativeDeposit();
      }
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "充值失败");
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

  const amountValue = depositAsset === "collateral" ? collateralAmount : nativeAmount;
  const amountStep = depositAsset === "collateral" ? ACCOUNTING_STEP : "0.001";
  const amountSymbol = depositAsset === "collateral" ? chain.tokenSymbol : chain.nativeCurrencySymbol;

  function onAmountChange(v: string) {
    if (depositAsset === "collateral") {
      setCollateralAmount(v);
    } else {
      setNativeAmount(v);
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <header className={styles.head}>
        <div>
          <span className="eyebrow">充值与提现</span>
          <p className={styles.sub}>链上资金进 Vault，入账后可在站内交易；下方分步操作。</p>
        </div>
        {walletLabel ? <span className="pill">{walletLabel}</span> : null}
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
          <p className={styles.tabHint}>
            {depositAsset === "collateral"
              ? `${chain.tokenSymbol}：未授权过 Vault 时会先无限额授权（签一次）再充值（再签一次）；之后通常只需签充值。`
              : `${chain.nativeCurrencySymbol}：按预言机价格折算为 ${chain.tokenSymbol} 记账额度，无需代币授权，一笔交易即可。`}
          </p>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>充值资产</span>
            <select
              className={styles.select}
              value={depositAsset}
              onChange={(e) => setDepositAsset(e.target.value as DepositAsset)}
              aria-label="选择充值资产"
            >
              <option value="collateral">{chain.tokenSymbol}（抵押代币）</option>
              <option value="native">{chain.nativeCurrencySymbol}（原生币，折算入账）</option>
            </select>
          </label>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>金额（{amountSymbol}）</span>
            <input
              className={styles.inputWide}
              value={amountValue}
              onChange={(e) => onAmountChange(e.target.value)}
              inputMode="decimal"
              step={amountStep}
            />
          </label>

          <button type="button" className={styles.primary} onClick={handleDepositSubmit} disabled={busy === "connect"}>
            {busy === "connect" ? "连接中…" : "充值"}
          </button>
        </div>
      ) : (
        <div className={styles.tabPanel} role="tabpanel">
          <p className={styles.tabHint}>从 Vault 发起提现链上请求；请核对收款地址。</p>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>金额（{chain.tokenSymbol}）</span>
            <input
              className={styles.inputWide}
              value={withdrawAmount}
              onChange={(e) => setWithdrawAmount(e.target.value)}
              inputMode="decimal"
              step={ACCOUNTING_STEP}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>收款地址</span>
            <input
              className={styles.inputWide}
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              placeholder={wallet?.walletAddress ?? "0x…"}
            />
          </label>
          <button type="button" className={styles.primary} onClick={handleWithdrawal} disabled={busy === "connect"}>
            提现
          </button>
        </div>
      )}

      {!wallet ? (
        <p className={styles.hint}>尚未连接钱包时，点击「充值」或「提现」会先请求连接。</p>
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
