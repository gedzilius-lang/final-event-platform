'use client'
import { useState } from 'react'
import dynamic from 'next/dynamic'

const StripePaymentModal = dynamic(() => import('@/components/StripePaymentModal'), { ssr: false })

interface TopupIntent {
  topup_id: string
  client_secret?: string
  redirect_url?: string
  amount_chf: number
  amount_nc: number
}

interface Props {
  amountChf: number
}

export default function TopUpButton({ amountChf }: Props) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [intent, setIntent] = useState<TopupIntent | null>(null)

  async function handleTopUp() {
    setError('')
    setLoading(true)

    const res = await fetch('/api/wallet/topup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount_chf: amountChf }),
    })

    if (!res.ok) {
      const data = await res.json()
      setError(data.error ?? 'Failed to start payment')
      setLoading(false)
      return
    }

    const data: TopupIntent = await res.json()
    setLoading(false)

    if (data.redirect_url) {
      // TWINT or other redirect-based provider
      window.location.href = data.redirect_url
    } else if (data.client_secret) {
      setIntent(data)
    } else {
      setError('Unexpected payment response from server')
    }
  }

  const ncAmount = amountChf * 100

  return (
    <>
      <div>
        <button
          onClick={handleTopUp}
          disabled={loading}
          className="btn-ghost w-full py-4 flex flex-col items-center gap-0.5"
        >
          <span className="text-lg font-bold text-nite-accent">+{amountChf} CHF</span>
          <span className="text-xs text-nite-muted">{ncAmount.toLocaleString()} NC</span>
          {loading && <span className="text-xs">…</span>}
        </button>
        {error && <p className="text-red-400 text-xs mt-1 text-center px-2">{error}</p>}
      </div>

      {intent?.client_secret && (
        <StripePaymentModal
          clientSecret={intent.client_secret}
          amountCHF={intent.amount_chf}
          amountNC={intent.amount_nc}
          onClose={() => setIntent(null)}
        />
      )}
    </>
  )
}
