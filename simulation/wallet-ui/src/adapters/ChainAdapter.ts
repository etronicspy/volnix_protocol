import { GasPrice, SigningStargateClient, StargateClient } from '@cosmjs/stargate'
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing'
import type { Account, Order, Transaction } from '../types'
import type {
  BackendAdapter,
  SubscribeCallbacks,
  TxResult,
  WalletState,
} from './types'

const MICRO = 1_000_000

export class ChainAdapter implements BackendAdapter {
  readonly mode = 'chain' as const

  private rpcUrl: string
  private restUrl: string
  private prefix: string
  private chainId: string

  private client: StargateClient | null = null
  private signingClient: SigningStargateClient | null = null
  private wallet: DirectSecp256k1HdWallet | null = null
  private address = ''

  constructor(
    rpcUrl: string,
    restUrl: string,
    prefix = 'volnix',
    chainId = 'volnix-1',
  ) {
    this.rpcUrl = rpcUrl
    this.restUrl = restUrl
    this.prefix = prefix
    this.chainId = chainId
  }

  async connectWithMnemonic(mnemonic: string): Promise<string> {
    this.wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: this.prefix,
    })
    const [{ address }] = await this.wallet.getAccounts()
    this.address = address

    this.client = await StargateClient.connect(this.rpcUrl)
    this.signingClient = await SigningStargateClient.connectWithSigner(
      this.rpcUrl,
      this.wallet,
      { gasPrice: GasPrice.fromString('0.025uwrt') },
    )

    const actual = await this.client.getChainId()
    if (actual !== this.chainId) {
      console.warn(`Chain ID mismatch: expected ${this.chainId}, got ${actual}`)
    }

    return address
  }

  private async queryAccount(): Promise<Account> {
    if (!this.client) throw new Error('Not connected')
    const balances = await this.client.getAllBalances(this.address)
    const toFloat = (denom: string): number => {
      const c = balances.find((b) => b.denom === denom)
      return c ? parseInt(c.amount, 10) / MICRO : 0
    }

    let role = 'citizen'
    let zkp = false
    try {
      const res = await fetch(
        `${this.restUrl}/volnix/ident/v1/identity/${this.address}`,
      )
      if (res.ok) {
        const data = (await res.json()) as {
          identity?: { role?: string; zkp_verified?: boolean }
        }
        role = data.identity?.role?.toLowerCase() || 'citizen'
        zkp = data.identity?.zkp_verified === true
      }
    } catch {
      /* module may not be deployed yet */
    }

    return {
      address: this.address,
      wrt_balance: toFloat('uwrt'),
      lzn_balance: toFloat('ulzn'),
      ant_balance: toFloat('uant'),
      role,
      zkp_verified: zkp,
    }
  }

  async getState(): Promise<WalletState> {
    if (!this.client) throw new Error('Not connected — import mnemonic first')
    const height = await this.client.getHeight()
    const account = await this.queryAccount()

    let market = null
    try {
      const res = await fetch(`${this.restUrl}/volnix/anteil/v1/orderbook`)
      if (res.ok) market = await res.json()
    } catch {
      /* module may not be deployed */
    }

    return {
      height,
      accounts_count: 1,
      accounts: { [this.address]: account },
      market,
    }
  }

  subscribe({ onState, onConnected, onError }: SubscribeCallbacks): () => void {
    let cancelled = false
    const poll = async () => {
      while (!cancelled) {
        try {
          const s = await this.getState()
          if (!cancelled) {
            onState(s)
            onConnected?.(true)
          }
        } catch (err) {
          if (!cancelled) {
            onConnected?.(false)
            onError?.(err instanceof Error ? err.message : 'Poll failed')
          }
        }
        await new Promise<void>((r) => setTimeout(r, 6000))
      }
    }
    void poll()
    return () => {
      cancelled = true
    }
  }

  async submit(body: Record<string, unknown>): Promise<TxResult> {
    const op = body.op as string

    if (op === 'transfer') {
      return this.doTransfer(
        body.to_address as string,
        body.amount as number,
        (body.asset as string) || 'wrt',
      )
    }

    const moduleOps = [
      'set_role',
      'verify_zkp',
      'activate_lzn',
      'declare',
      'create_order',
      'cancel_order',
    ]
    if (moduleOps.includes(op)) {
      return {
        accepted: false,
        message: `Операция «${op}» требует развёрнутых Volnix-модулей (x/ident, x/lizenz, x/anteil). Используйте режим симуляции для тестирования.`,
      }
    }

    return { accepted: false, message: `Unknown op: ${op}` }
  }

  private async doTransfer(
    to: string,
    amount: number,
    asset: string,
  ): Promise<TxResult> {
    if (!this.signingClient || !this.address) {
      return { accepted: false, message: 'Wallet not connected' }
    }
    const denom = asset.startsWith('u') ? asset : `u${asset}`
    const microAmt = String(Math.floor(amount * MICRO))
    try {
      const result = await this.signingClient.sendTokens(
        this.address,
        to,
        [{ denom, amount: microAmt }],
        'auto',
      )
      if (result.code !== 0) {
        return {
          accepted: false,
          message: `Tx failed (code ${result.code}): ${result.rawLog}`,
        }
      }
      return {
        accepted: true,
        message: 'Broadcast OK',
        tx_hash: result.transactionHash,
      }
    } catch (err) {
      return {
        accepted: false,
        message: err instanceof Error ? err.message : 'Broadcast error',
      }
    }
  }

  async fetchHistory(
    address: string,
  ): Promise<{ history: Transaction[]; open_orders: Order[] }> {
    if (!this.client) return { history: [], open_orders: [] }
    try {
      const txs = await this.client.searchTx(
        `message.sender='${address}'`,
      )
      const history: Transaction[] = txs.map((tx) => ({
        tx_hash: tx.hash,
        tx_type: 'transfer',
        sender: address,
        receiver: '',
        amount: 0,
        price: 0,
        role: '',
        timestamp: 0,
        details: `Height ${tx.height}`,
      }))
      return { history, open_orders: [] }
    } catch {
      return { history: [], open_orders: [] }
    }
  }

  async fetchOpenOrders(): Promise<Order[]> {
    return []
  }

  disconnect(): void {
    this.client?.disconnect()
    this.client = null
    this.signingClient?.disconnect()
    this.signingClient = null
    this.wallet = null
    this.address = ''
  }
}
