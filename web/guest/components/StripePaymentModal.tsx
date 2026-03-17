'use client'
import { useEffect, useState } from 'react'
import { loadStripe, Stripe } from '@stripe/stripe-js'
import {
  Elements,
  PaymentElement,
  useStripe,
  useElements,
} from '@stripe/react-stripe-js'

interface Props {
  clientSecret: string
  amountCHF: number
  amountNC: number
  onClose: () => void
}

// Load Stripe outside component to avoid re-instantiation
let stripePromise: ReturnType<typeof loadStripe> | null = null
function getStripe() {
  if (!stripePromise) {
    const pk = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
    if (!pk) throw new Error('NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY not set')
    stripePromise = loadStripe(pk)
  }
  return stripePromise
}

export default function StripePaymentModal({ clientSecret, amountCHF, amountNC, onClose }: Props) {
  const appearance = {
    theme: 'night' as const,
    variables: {
      colorPrimary: '#f59e0b',
      colorBackground: '#141414',
      colorText: '#f5f5f5',
      colorDanger: '#ef4444',
      borderRadius: '8px',
      fontFamily: 'system-ui, sans-serif',
    },
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="bg-nite-surface border border-nite-border rounded-t-2xl sm:rounded-2xl w-full max-w-md p-6 pb-8">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="font-bold text-lg">Top up wallet</h2>
            <p className="text-sm text-nite-muted">
              {amountCHF} CHF → {amountNC.toLocaleString()} NC
            </p>
          </div>
          <button onClick={onClose} className="text-nite-muted hover:text-nite-text text-2xl leading-none">×</button>
        </div>

        <Elements stripe={getStripe()} options={{ clientSecret, appearance }}>
          <CheckoutForm amountCHF={amountCHF} onClose={onClose} />
        </Elements>
      </div>
    </div>
  )
}

function CheckoutForm({ amountCHF, onClose }: { amountCHF: number; onClose: () => void }) {
  const stripe = useStripe()
  const elements = useElements()
  const [loading, setLoading] = useState(false)
  const [errorMsg, setErrorMsg] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!stripe || !elements) return

    setLoading(true)
    setErrorMsg('')

    const returnUrl = `${window.location.origin}/wallet/success`

    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: { return_url: returnUrl },
    })

    // confirmPayment only returns an error if it couldn't redirect
    if (error) {
      setErrorMsg(error.message ?? 'Payment failed. Please try again.')
      setLoading(false)
    }
    // On success, Stripe redirects to return_url — this code won't run
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <PaymentElement
        options={{
          layout: 'tabs',
          defaultValues: { billingDetails: { address: { country: 'CH' } } },
        }}
      />

      {errorMsg && (
        <p className="text-red-400 text-sm">{errorMsg}</p>
      )}

      <button
        type="submit"
        disabled={!stripe || !elements || loading}
        className="btn-primary w-full py-3"
      >
        {loading ? 'Processing…' : `Pay CHF ${amountCHF.toFixed(2)}`}
      </button>

      <p className="text-xs text-nite-muted text-center">
        Secured by Stripe · Your card details are never stored by NiteOS
      </p>
    </form>
  )
}
