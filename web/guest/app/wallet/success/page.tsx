// Stripe redirects here after payment: /wallet/success?payment_intent=pi_...&redirect_status=succeeded
import { redirect } from 'next/navigation'

interface Props {
  searchParams: { redirect_status?: string; payment_intent?: string }
}

export default function WalletSuccessPage({ searchParams }: Props) {
  // Forward query params to wallet page which shows the success banner
  const status = searchParams.redirect_status ?? 'unknown'
  if (status === 'succeeded') {
    redirect('/wallet?redirect_status=succeeded')
  }
  // Failed or cancelled
  redirect('/wallet?redirect_status=failed')
}
