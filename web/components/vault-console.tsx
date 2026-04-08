"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { approveVault, depositNativeToVault, depositToVault, ensureTargetChain, getChainMeta, queueWithdrawal } from "@/lib/chain";
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

export function VaultConsole() {
  const { wallet, session, busy, connect } = useTradingSession();
  const chain = useMemo(() => getChainMeta(), []);
  const [depositAmount, setDepositAmount] = useState("100");
  const [nativeDepositAmount, setNativeDepositAmount] = useState("0.1");
  const [withdrawAmount, setWithdrawAmount] = useState("50");
  const [recipient, setRecipient] = useState("");
  const [status, setStatus] = useState("充值通道待命");
  

  async function handleConnect() {
    setStatus("连接钱包中...");
    try {
      await connect();
      setStatus("钱包已连接，请先授权，再充值到 Vault。");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "钱包连接失败");
    }
  }

  async function handleApprove() {
    if (!wallet) {
      setStatus("请先连接钱包。");
      return;
    }
    setStatus("正在切换网络并发起授权...");
    try {
      const normalizedAmount = validateAmountInput(depositAmount, chain.tokenSymbol);
      await ensureTargetChain();
      const txHash = await approveVault(wallet.walletAddress, normalizedAmount);
      setStatus(`授权已提交：${txHash}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "授权失败");
    }
  }

  async function handleDeposit() {
    if (!wallet) {
      setStatus("请先连接钱包。");
      return;
    }
    setStatus("正在发起充值交易...");
    try {
      const normalizedAmount = validateAmountInput(depositAmount, chain.tokenSymbol);
      await ensureTargetChain();
      const txHash = await depositToVault(wallet.walletAddress, normalizedAmount);
      setStatus(`充值交易已提交：${txHash}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "充值失败");
    }
  }

  async function handleNativeDeposit() {
    if (!wallet) {
      setStatus("请先连接钱包。");
      return;
    }
    const trimmed = nativeDepositAmount.trim();
    if (!trimmed || !/^\d+(\.\d+)?$/.test(trimmed) || Number(trimmed) <= 0) {
      setStatus(`请输入有效的 ${chain.nativeCurrencySymbol} 金额。`);
      return;
    }
    setStatus(`正在发起 ${chain.nativeCurrencySymbol} 充值交易...`);
    try {
      await ensureTargetChain();
      const txHash = await depositNativeToVault(wallet.walletAddress, trimmed);
      setStatus(`${chain.nativeCurrencySymbol} 充值已提交：${txHash}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : `${chain.nativeCurrencySymbol} 充值失败`);
    }
  }

  async function handleWithdrawal() {
    if (!wallet) {
      setStatus("请先连接钱包。");
      return;
    }
    if (!recipient.trim()) {
      setStatus("请输入收款地址。");
      return;
    }
    setStatus("正在发起提现请求...");
    try {
      const normalizedAmount = validateAmountInput(withdrawAmount, chain.tokenSymbol);
      await ensureTargetChain();
      const { txHash, withdrawalId } = await queueWithdrawal(wallet.walletAddress, normalizedAmount, recipient.trim());
      setStatus(`提现请求已提交：${txHash} / ${withdrawalId}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "提现失败");
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">充值与提现</span>
          <p className={styles.copy}>连接钱包后，可用 {chain.nativeCurrencySymbol} 一键充值（自动换算为 {chain.tokenSymbol}），或使用 {chain.tokenSymbol} 走标准授权 + 充值流程。</p>
        </div>
        <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "钱包未连接"}</span>
      </div>

      <div className={styles.guide}>
        <article className={styles.guideCard}>
          <span className={styles.stepNo}>01</span>
          <div>
            <strong>连接钱包</strong>
            <p>页面不会自动弹钱包。只有在你点击后才会发起连接。</p>
          </div>
        </article>
        <article className={styles.guideCard}>
          <span className={styles.stepNo}>02</span>
          <div>
            <strong>授权 {chain.tokenSymbol}</strong>
            <p>首次授权后，Vault 合约才能转走代币。通常每次 allowance 重置后再授权一次即可。</p>
          </div>
        </article>
        <article className={styles.guideCard}>
          <span className={styles.stepNo}>03</span>
          <div>
            <strong>充值并等待入账</strong>
            <p>后端会监听 Vault 事件，并把到账金额同步到 FunnyOption 余额里。</p>
          </div>
        </article>
      </div>

      <div className={styles.metaGrid}>
        <div className={styles.metaCard}>
          <span className={styles.label}>网络</span>
          <span className={styles.metaValue}>{chain.chainName} / {chain.chainId}</span>
        </div>
        <div className={styles.metaCard}>
          <span className={styles.label}>Vault</span>
          <span className={styles.metaValue}>{chain.vaultAddress || "请先配置 NEXT_PUBLIC_VAULT_ADDRESS"}</span>
        </div>
        <div className={styles.metaCard}>
          <span className={styles.label}>入账路径</span>
          <span className={styles.metaValue}>{session ? "钱包 → Vault → 监听器 → 余额" : "连接钱包后开始"}</span>
        </div>
      </div>

      <div className={styles.grid}>
        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <span className="pill">充值</span>
            <h3 className={styles.title}>充值到交易余额</h3>
            <p className={styles.helper}>先授权，再充值。监听器完成入账后，可用 USDT 会更新。金额最多支持 {ACCOUNTING_DECIMALS} 位小数。</p>
          </div>
          <label className={styles.field}>
            <span className={styles.label}>金额（{chain.tokenSymbol}）</span>
            <input className={styles.input} value={depositAmount} onChange={(event) => setDepositAmount(event.target.value)} inputMode="decimal" step={ACCOUNTING_STEP} />
          </label>
          <div className={styles.actions}>
            {!wallet ? (
              <button className={styles.ghost} onClick={handleConnect}>
                {busy === "connect" ? "连接中..." : "连接钱包"}
              </button>
            ) : null}
            <button className={styles.ghost} onClick={handleApprove}>授权</button>
            <button className={styles.button} onClick={handleDeposit}>充值</button>
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <span className="pill">{chain.nativeCurrencySymbol} 充值</span>
            <h3 className={styles.title}>用 {chain.nativeCurrencySymbol} 充值</h3>
            <p className={styles.helper}>
              直接发送 {chain.nativeCurrencySymbol}，系统通过 Chainlink 预言机实时计算等值 {chain.tokenSymbol} 并入账。无需授权，一笔交易完成。
            </p>
          </div>
          <label className={styles.field}>
            <span className={styles.label}>金额（{chain.nativeCurrencySymbol}）</span>
            <input className={styles.input} value={nativeDepositAmount} onChange={(event) => setNativeDepositAmount(event.target.value)} inputMode="decimal" step="0.001" />
          </label>
          <div className={styles.actions}>
            {!wallet ? (
              <button className={styles.ghost} onClick={handleConnect}>
                {busy === "connect" ? "连接中..." : "连接钱包"}
              </button>
            ) : null}
            <button className={styles.button} onClick={handleNativeDeposit}>{chain.nativeCurrencySymbol} 充值</button>
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <span className="pill">提现</span>
            <h3 className={styles.title}>发起提现</h3>
            <p className={styles.helper}>提现同样会直达 Vault。填写收款地址并签名后，请求会记录到链上。金额最多支持 {ACCOUNTING_DECIMALS} 位小数。</p>
          </div>
          <label className={styles.field}>
            <span className={styles.label}>金额（{chain.tokenSymbol}）</span>
            <input className={styles.input} value={withdrawAmount} onChange={(event) => setWithdrawAmount(event.target.value)} inputMode="decimal" step={ACCOUNTING_STEP} />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>收款地址</span>
            <input
              className={styles.input}
              value={recipient}
              onChange={(event) => setRecipient(event.target.value)}
              placeholder={wallet?.walletAddress ?? "0x..."}
            />
          </label>
          <div className={styles.actions}>
            {!wallet ? (
              <button className={styles.ghost} onClick={handleConnect}>
                {busy === "connect" ? "连接中..." : "连接钱包"}
              </button>
            ) : null}
            <button className={styles.button} onClick={handleWithdrawal}>提现</button>
          </div>
        </div>
      </div>

      <div className={styles.status}>{status}</div>
    </section>
  );
}
