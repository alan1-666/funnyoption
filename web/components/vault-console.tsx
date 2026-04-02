"use client";

import { useMemo, useState } from "react";

import { useTradingSession } from "@/components/trading-session-provider";
import { approveVault, depositToVault, ensureTargetChain, getChainMeta, queueWithdrawal } from "@/lib/chain";
import { shortenAddress } from "@/lib/format";
import styles from "@/components/vault-console.module.css";

export function VaultConsole() {
  const { wallet, session, busy, connect } = useTradingSession();
  const chain = useMemo(() => getChainMeta(), []);
  const [depositAmount, setDepositAmount] = useState("100");
  const [withdrawAmount, setWithdrawAmount] = useState("50");
  const [recipient, setRecipient] = useState("");
  const [status, setStatus] = useState("Wallet vault lane idle");

  async function handleConnect() {
    setStatus("Connecting wallet...");
    try {
      await connect();
      setStatus("Wallet linked. Approve once, then deposit to the vault.");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Wallet connection failed");
    }
  }

  async function handleApprove() {
    if (!wallet) {
      setStatus("Connect wallet first.");
      return;
    }
    setStatus("Switching chain and opening approval...");
    try {
      await ensureTargetChain();
      const txHash = await approveVault(wallet.walletAddress, depositAmount);
      setStatus(`Approve submitted: ${txHash}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Approve failed");
    }
  }

  async function handleDeposit() {
    if (!wallet) {
      setStatus("Connect wallet first.");
      return;
    }
    setStatus("Opening deposit transaction...");
    try {
      await ensureTargetChain();
      const txHash = await depositToVault(wallet.walletAddress, depositAmount);
      setStatus(`Deposit submitted: ${txHash}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Deposit failed");
    }
  }

  async function handleWithdrawal() {
    if (!wallet) {
      setStatus("Connect wallet first.");
      return;
    }
    if (!recipient.trim()) {
      setStatus("Recipient address is required.");
      return;
    }
    setStatus("Opening withdrawal request...");
    try {
      await ensureTargetChain();
      const { txHash, withdrawalId } = await queueWithdrawal(wallet.walletAddress, withdrawAmount, recipient.trim());
      setStatus(`Withdrawal submitted: ${txHash} / ${withdrawalId}`);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Withdrawal failed");
    }
  }

  return (
    <section className={`panel ${styles.console}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">Vault lane</span>
          <p className={styles.copy}>Recharge now follows one clear path: connect wallet, approve the token, deposit into the vault, then wait for the deposit listener to credit your trading balance.</p>
        </div>
        <span className="pill">{wallet ? shortenAddress(wallet.walletAddress) : "Wallet offline"}</span>
      </div>

      <div className={styles.guide}>
        <article className={styles.guideCard}>
          <span className={styles.stepNo}>01</span>
          <div>
            <strong>Connect wallet</strong>
            <p>We do not auto-touch the wallet on page load anymore. Connection starts only after your click.</p>
          </div>
        </article>
        <article className={styles.guideCard}>
          <span className={styles.stepNo}>02</span>
          <div>
            <strong>Approve {chain.tokenSymbol}</strong>
            <p>The first approval lets the vault contract move the token. Usually you only need to do this once per allowance reset.</p>
          </div>
        </article>
        <article className={styles.guideCard}>
          <span className={styles.stepNo}>03</span>
          <div>
            <strong>Deposit and wait for credit</strong>
            <p>The backend listens for the vault event and mirrors the credited amount into your FunnyOption balance.</p>
          </div>
        </article>
      </div>

      <div className={styles.metaGrid}>
        <div className={styles.metaCard}>
          <span className={styles.label}>Network</span>
          <span className={styles.metaValue}>{chain.chainName} / {chain.chainId}</span>
        </div>
        <div className={styles.metaCard}>
          <span className={styles.label}>Vault</span>
          <span className={styles.metaValue}>{chain.vaultAddress || "Set NEXT_PUBLIC_VAULT_ADDRESS"}</span>
        </div>
        <div className={styles.metaCard}>
          <span className={styles.label}>Credit path</span>
          <span className={styles.metaValue}>{session ? "Wallet → Vault → Listener → Balance" : "Connect wallet to start"}</span>
        </div>
      </div>

      <div className={styles.grid}>
        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <span className="pill">Top up</span>
            <h3 className={styles.title}>Deposit into trading balance</h3>
            <p className={styles.helper}>Approve first, then deposit. Your available USDT updates after the deposit listener credits the vault event.</p>
          </div>
          <label className={styles.field}>
            <span className={styles.label}>Amount ({chain.tokenSymbol})</span>
            <input className={styles.input} value={depositAmount} onChange={(event) => setDepositAmount(event.target.value)} />
          </label>
          <div className={styles.actions}>
            {!wallet ? (
              <button className={styles.ghost} onClick={handleConnect}>
                {busy === "connect" ? "Connecting..." : "Connect Wallet"}
              </button>
            ) : null}
            <button className={styles.ghost} onClick={handleApprove}>Approve</button>
            <button className={styles.button} onClick={handleDeposit}>Deposit</button>
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <span className="pill">Cash out</span>
            <h3 className={styles.title}>Request withdrawal</h3>
            <p className={styles.helper}>Withdrawals are also direct vault actions. Pick a recipient, sign once, and the request is recorded on-chain.</p>
          </div>
          <label className={styles.field}>
            <span className={styles.label}>Amount ({chain.tokenSymbol})</span>
            <input className={styles.input} value={withdrawAmount} onChange={(event) => setWithdrawAmount(event.target.value)} />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>Recipient</span>
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
                {busy === "connect" ? "Connecting..." : "Connect Wallet"}
              </button>
            ) : null}
            <button className={styles.button} onClick={handleWithdrawal}>Withdraw</button>
          </div>
        </div>
      </div>

      <div className={styles.status}>{status}</div>
    </section>
  );
}
