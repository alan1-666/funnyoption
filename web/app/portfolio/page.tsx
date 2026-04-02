import { ClaimConsole } from "@/components/claim-console";
import { SessionConsole } from "@/components/session-console";
import { SiteHeader } from "@/components/site-header";
import { VaultConsole } from "@/components/vault-console";
import { formatTimestamp, formatToken, shortenAddress } from "@/lib/format";
import { getBalances, getDeposits, getPayouts, getPositions, getSessions, getWithdrawals } from "@/lib/api";
import styles from "@/app/portfolio/page.module.css";

export default async function PortfolioPage() {
  const [balances, positions, deposits, withdrawals, payouts, sessions] = await Promise.all([
    getBalances(),
    getPositions(),
    getDeposits(),
    getWithdrawals(),
    getPayouts(),
    getSessions()
  ]);

  const usdt = balances.find((item) => item.asset === "USDT");
  const totalPositions = positions.reduce((sum, position) => sum + position.quantity, 0);
  const pendingPayoutValue = payouts.reduce((sum, payout) => sum + payout.payout_amount, 0);

  return (
    <main className="page-shell">
      <SiteHeader />

      <section className={styles.hero}>
        <div className={`panel ${styles.heroMain} float-in`}>
          <span className="eyebrow">Portfolio</span>
          <h1 className={styles.title}>Balances, positions, and payout activity.</h1>
          <p className={styles.copy}>
            Track available balance, open positions, deposits, withdrawals, and claimable payouts in one place.
          </p>
        </div>

        <div className={`${styles.heroRail} float-in float-in-delay-1`}>
          <div className={`panel ${styles.metric}`}>
            <span className={styles.label}>Available USDT</span>
            <strong>{formatToken(usdt?.available ?? 0, 0)}</strong>
          </div>
          <div className={`panel ${styles.metric}`}>
            <span className={styles.label}>Frozen USDT</span>
            <strong>{formatToken(usdt?.frozen ?? 0, 0)}</strong>
          </div>
          <div className={`panel ${styles.metric}`}>
            <span className={styles.label}>Open positions</span>
            <strong>{formatToken(totalPositions, 0)}</strong>
          </div>
          <div className={`panel ${styles.metric}`}>
            <span className={styles.label}>Payout inventory</span>
            <strong>{formatToken(pendingPayoutValue, 0)}</strong>
          </div>
        </div>
      </section>

      <div className="float-in float-in-delay-1">
        <VaultConsole />
      </div>

      <div className="float-in float-in-delay-2">
        <SessionConsole initialSessions={sessions} />
      </div>

      <section className={styles.grid}>
        <section className={`panel ${styles.block} float-in float-in-delay-2`}>
          <div className={styles.blockHeader}>
            <div>
              <span className="eyebrow">Balances</span>
              <h2 className={styles.blockTitle}>Asset state</h2>
            </div>
          </div>
          <div className={styles.balanceList}>
            {balances.map((balance) => (
              <article key={balance.asset} className={styles.balanceCard}>
                <span className={styles.label}>{balance.asset}</span>
                <strong className={styles.balanceValue}>{formatToken(balance.available, 0)}</strong>
                <div className={styles.balanceMeta}>
                  <span>Frozen {formatToken(balance.frozen, 0)}</span>
                  <span>Updated {formatTimestamp(balance.updated_at)}</span>
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className={`panel ${styles.block} float-in float-in-delay-2`}>
          <div className={styles.blockHeader}>
            <div>
              <span className="eyebrow">Positions</span>
              <h2 className={styles.blockTitle}>Exposure</h2>
            </div>
          </div>
          <div className={styles.positionList}>
            {positions.map((position) => (
              <div key={`${position.market_id}-${position.outcome}`} className={styles.positionRow}>
                <div>
                  <div className={styles.positionTitle}>Market #{position.market_id} · {position.outcome}</div>
                  <div className={styles.positionMeta}>{position.position_asset}</div>
                </div>
                <div className={styles.positionQty}>
                  <strong>{formatToken(position.quantity, 0)}</strong>
                  <span>settled {formatToken(position.settled_quantity, 0)}</span>
                </div>
              </div>
            ))}
          </div>
        </section>
      </section>

      <section className={styles.lowerRow}>
        <section className={`panel ${styles.block} float-in float-in-delay-3`}>
          <div className={styles.blockHeader}>
            <div>
              <span className="eyebrow">Direct deposits</span>
              <h2 className={styles.blockTitle}>Vault credits</h2>
            </div>
          </div>
          <div className={styles.depositList}>
            {deposits.map((deposit) => (
              <div key={deposit.deposit_id} className={styles.depositRow}>
                <div>
                  <div className={styles.positionTitle}>{formatToken(deposit.amount, 0)} {deposit.asset}</div>
                  <div className={styles.positionMeta}>
                    {shortenAddress(deposit.wallet_address)} → {shortenAddress(deposit.vault_address)}
                  </div>
                </div>
                <div className={styles.depositMeta}>
                  <strong>{deposit.status}</strong>
                  <span>{formatTimestamp(deposit.credited_at || deposit.created_at)}</span>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className={`panel ${styles.block} float-in float-in-delay-3`}>
          <div className={styles.blockHeader}>
            <div>
              <span className="eyebrow">Queued withdrawals</span>
              <h2 className={styles.blockTitle}>Vault debits</h2>
            </div>
          </div>
          <div className={styles.depositList}>
            {withdrawals.map((withdrawal) => (
              <div key={withdrawal.withdrawal_id} className={styles.depositRow}>
                <div>
                  <div className={styles.positionTitle}>{formatToken(withdrawal.amount, 0)} {withdrawal.asset}</div>
                  <div className={styles.positionMeta}>
                    {shortenAddress(withdrawal.wallet_address)} → {shortenAddress(withdrawal.recipient_address)}
                  </div>
                </div>
                <div className={styles.depositMeta}>
                  <strong>{withdrawal.status}</strong>
                  <span>{formatTimestamp(withdrawal.debited_at || withdrawal.created_at)}</span>
                </div>
              </div>
            ))}
          </div>
        </section>

        <div className="float-in float-in-delay-3">
          <ClaimConsole payouts={payouts} deposits={deposits} />
        </div>
      </section>

    </main>
  );
}
