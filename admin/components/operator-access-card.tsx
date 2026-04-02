"use client";

import styles from "@/components/operator-access-card.module.css";
import { shortenAddress } from "@/lib/format";
import { useOperatorAccess } from "@/components/operator-access-provider";

export function OperatorAccessCard() {
  const { wallet, busy, statusMessage, allowlistedWallets, isWalletAllowlisted, connect } = useOperatorAccess();
  const hasAllowlist = allowlistedWallets.length > 0;
  const targetChainId = Number(process.env.NEXT_PUBLIC_CHAIN_ID ?? "97");
  const targetChainName = process.env.NEXT_PUBLIC_CHAIN_NAME ?? "BSC Testnet";
  const isTargetChain = wallet ? wallet.chainId === targetChainId : false;
  const hasOperatorAccess = hasAllowlist && !!wallet && isWalletAllowlisted && isTargetChain;
  const actionLabel =
    busy === "connect"
      ? "连接中..."
      : wallet && !isTargetChain
        ? `切换到 ${targetChainName}`
        : wallet
          ? "重新连接钱包"
          : "连接钱包";

  return (
    <section className={`panel ${styles.panel}`}>
      <div className={styles.header}>
        <div>
          <span className="eyebrow">运营身份</span>
          <h2 className={styles.title}>先确认钱包身份，再执行运营动作。</h2>
          <p className={styles.copy}>创建市场、发首发流动性和结算都走同一条签名通道。白名单和目标链通过后，后台动作才会开放。</p>
        </div>
        <div className={styles.badges}>
          <span className="pill">{hasAllowlist ? `白名单 ${allowlistedWallets.length} 个` : "未配置白名单"}</span>
          <span className="pill">{wallet ? `链 ${wallet.chainId}` : "钱包未连接"}</span>
        </div>
      </div>

      <div className={styles.grid}>
        <div className={styles.identityCard}>
          <span className={styles.label}>当前钱包</span>
          <strong className={styles.value}>{wallet ? shortenAddress(wallet.walletAddress) : "未连接"}</strong>
          <p className={styles.meta}>{wallet ? wallet.walletAddress : "连接运营钱包后，后台动作才会开放。"}</p>
        </div>

        <div className={styles.identityCard}>
          <span className={styles.label}>权限状态</span>
          <strong className={hasOperatorAccess ? styles.valueOk : styles.valueDanger}>
            {!hasAllowlist ? "不可用" : hasOperatorAccess ? "已允许" : "未通过"}
          </strong>
          <p className={styles.meta}>
            {!hasAllowlist
              ? "请先配置 FUNNYOPTION_OPERATOR_WALLETS。"
              : wallet && isWalletAllowlisted && isTargetChain
                ? "当前钱包可以在独立后台里执行创建市场、首发流动性和结算。"
                : wallet && isWalletAllowlisted
                  ? `当前钱包已在白名单中，但需要切换到 ${targetChainName}（${targetChainId}）。`
                : "非白名单钱包只能查看读面，不能执行运营动作。"}
          </p>
        </div>
      </div>

      <div className={styles.actions}>
        <button className={styles.primary} type="button" disabled={busy === "connect"} onClick={() => connect()}>
          {actionLabel}
        </button>
        <div className={styles.status} aria-live="polite">
          {statusMessage}
        </div>
      </div>

      {allowlistedWallets.length > 0 ? (
        <div className={styles.allowlist}>
          {allowlistedWallets.map((entry) => (
            <span key={entry} className="pill">{shortenAddress(entry)}</span>
          ))}
        </div>
      ) : null}
    </section>
  );
}
