'use client'
// Bar staff POS — client component.
// Staff selects items, enters guest NiteTap UID, confirms charge.
import { useState, useCallback } from 'react'

interface CatalogItem {
  item_id: string
  name: string
  category: string
  price_nc: number
  icon?: string
}

interface CartItem extends CatalogItem {
  qty: number
}

interface Props {
  items: CatalogItem[]
  venueId: string
}

type Step = 'cart' | 'identify' | 'confirm' | 'done'

export default function BarPOS({ items, venueId }: Props) {
  const [cart, setCart] = useState<Record<string, CartItem>>({})
  const [step, setStep] = useState<Step>('cart')
  const [guestUid, setGuestUid] = useState('')
  const [guestUserId, setGuestUserId] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [lastOrderId, setLastOrderId] = useState('')

  const cartItems = Object.values(cart)
  const totalNC = cartItems.reduce((s, i) => s + i.price_nc * i.qty, 0)

  const addItem = useCallback((item: CatalogItem) => {
    setCart((prev) => {
      const existing = prev[item.item_id]
      return {
        ...prev,
        [item.item_id]: existing
          ? { ...existing, qty: existing.qty + 1 }
          : { ...item, qty: 1 },
      }
    })
  }, [])

  const removeItem = useCallback((itemId: string) => {
    setCart((prev) => {
      const existing = prev[itemId]
      if (!existing) return prev
      if (existing.qty <= 1) {
        const next = { ...prev }
        delete next[itemId]
        return next
      }
      return { ...prev, [itemId]: { ...existing, qty: existing.qty - 1 } }
    })
  }, [])

  async function lookupGuest(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      // Look up by NiteTap UID via profiles service (best-effort — may 404 for anonymous)
      const res = await fetch(`/api/staff/guest-lookup-uid?uid=${encodeURIComponent(guestUid.trim())}`)
      if (res.ok) {
        const data = await res.json()
        setGuestUserId(data.user_id ?? '')
      }
      // If not found, guestUserId remains empty — order will be anonymous session
      setStep('confirm')
    } catch {
      setError('Lookup failed — proceed with anonymous session')
      setStep('confirm')
    } finally {
      setLoading(false)
    }
  }

  async function submitOrder() {
    setError('')
    setLoading(true)
    const iKey = `bar:${venueId}:${crypto.randomUUID()}`
    try {
      // Create order
      const createRes = await fetch('/api/staff/order', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          venue_id: venueId,
          items: cartItems.map((i) => ({
            item_id: i.item_id,
            name: i.name,
            quantity: i.qty,
            price_nc: i.price_nc,
          })),
          idempotency_key: iKey,
        }),
      })
      if (!createRes.ok) {
        const d = await createRes.json()
        setError(d.error ?? 'Order failed')
        return
      }
      const { order_id } = await createRes.json()

      // Finalize (charge)
      const finalRes = await fetch('/api/staff/order/finalize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_id, guest_user_id: guestUserId }),
      })
      if (!finalRes.ok) {
        const d = await finalRes.json()
        setError(d.error ?? 'Charge failed — check balance')
        return
      }
      setLastOrderId(order_id)
      setStep('done')
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  function reset() {
    setCart({})
    setStep('cart')
    setGuestUid('')
    setGuestUserId('')
    setError('')
    setLastOrderId('')
  }

  const categories = Array.from(new Set(items.map((i) => i.category)))

  if (step === 'done') {
    return (
      <div className="card text-center py-10 space-y-4">
        <span className="text-5xl">✓</span>
        <div>
          <p className="text-xl font-bold text-nite-accent">{totalNC} NC</p>
          <p className="text-nite-muted text-sm">Charged successfully</p>
          <p className="text-xs font-mono text-nite-muted mt-1">
            Order: {lastOrderId.slice(0, 8)}…
          </p>
        </div>
        <button onClick={reset} className="btn-primary px-8">
          New order
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* Cart summary — always visible */}
      {cartItems.length > 0 && (
        <div className="card border-nite-accent/30 bg-amber-950/10">
          <div className="flex items-center justify-between mb-2">
            <p className="text-xs font-semibold text-nite-accent uppercase tracking-wider">
              Order
            </p>
            <p className="text-lg font-black text-nite-accent">{totalNC} NC</p>
          </div>
          <div className="space-y-1">
            {cartItems.map((i) => (
              <div key={i.item_id} className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => removeItem(i.item_id)}
                    className="w-5 h-5 rounded-full border border-nite-border text-xs flex items-center justify-center text-nite-muted hover:text-red-400 hover:border-red-900"
                  >
                    −
                  </button>
                  <span>{i.name}</span>
                  <span className="text-nite-muted">×{i.qty}</span>
                </div>
                <span className="text-nite-muted text-xs">{i.price_nc * i.qty} NC</span>
              </div>
            ))}
          </div>
          {step === 'cart' && (
            <button
              onClick={() => setStep('identify')}
              className="btn-primary w-full mt-3"
            >
              Charge {totalNC} NC →
            </button>
          )}
        </div>
      )}

      {/* Menu */}
      {step === 'cart' && (
        <div className="space-y-4">
          {items.length === 0 ? (
            <p className="text-nite-muted text-sm text-center py-6">
              No menu items configured.
            </p>
          ) : (
            categories.map((cat) => (
              <div key={cat}>
                <h3 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-2 capitalize">
                  {cat}
                </h3>
                <div className="grid grid-cols-2 gap-2">
                  {items
                    .filter((i) => i.category === cat)
                    .map((item) => {
                      const qty = cart[item.item_id]?.qty ?? 0
                      return (
                        <button
                          key={item.item_id}
                          onClick={() => addItem(item)}
                          className="card flex flex-col items-center py-4 gap-1 hover:border-nite-accent/50 active:scale-95 transition-all relative"
                        >
                          {qty > 0 && (
                            <span className="absolute top-2 right-2 text-xs bg-nite-accent text-black font-bold rounded-full w-5 h-5 flex items-center justify-center">
                              {qty}
                            </span>
                          )}
                          {item.icon && <span className="text-2xl">{item.icon}</span>}
                          <span className="text-sm font-medium text-center">{item.name}</span>
                          <span className="text-xs font-bold text-nite-accent">
                            {item.price_nc} NC
                          </span>
                        </button>
                      )
                    })}
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {/* Guest identification step */}
      {step === 'identify' && (
        <div className="card space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-nite-muted">
            Identify Guest
          </h2>
          <form onSubmit={lookupGuest} className="space-y-3">
            <div>
              <label className="label">NiteTap UID</label>
              <input
                type="text"
                value={guestUid}
                onChange={(e) => setGuestUid(e.target.value)}
                className="input font-mono"
                placeholder="Guest taps wristband…"
                autoComplete="off"
                autoFocus
              />
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setStep('cart')}
                className="btn-ghost flex-1"
              >
                Back
              </button>
              <button type="submit" disabled={loading || !guestUid.trim()} className="btn-primary flex-1">
                {loading ? 'Looking up…' : 'Confirm →'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Confirm + charge step */}
      {step === 'confirm' && (
        <div className="card space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-nite-muted">
            Confirm charge
          </h2>
          <div className="bg-nite-bg border border-nite-border rounded-lg p-3">
            <p className="text-2xl font-black text-nite-accent text-center">{totalNC} NC</p>
            {guestUid && (
              <p className="text-xs font-mono text-center text-nite-muted mt-1">
                UID: {guestUid}
              </p>
            )}
          </div>
          <div className="flex gap-2">
            <button onClick={() => setStep('identify')} className="btn-ghost flex-1">
              Back
            </button>
            <button onClick={submitOrder} disabled={loading} className="btn-primary flex-1">
              {loading ? 'Charging…' : `Charge ${totalNC} NC`}
            </button>
          </div>
        </div>
      )}

      {error && <p className="text-red-400 text-sm text-center">{error}</p>}
    </div>
  )
}
