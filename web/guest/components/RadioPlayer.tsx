'use client'
// Global persistent radio player — rendered in root layout, fixed at bottom.
// Survives client-side navigation (App Router layout never unmounts).
// Streams from NEXT_PUBLIC_RADIO_STREAM_URL; metadata from AzuraCast API.
import { useState, useEffect, useRef } from 'react'

const STREAM_URL =
  process.env.NEXT_PUBLIC_RADIO_STREAM_URL ?? 'https://stream.peoplewelike.club/stream'
const RADIO_API_URL =
  process.env.NEXT_PUBLIC_RADIO_API_URL ?? 'https://radio.peoplewelike.club'

interface NowPlaying {
  title: string
  artist: string
}

async function fetchNowPlaying(): Promise<NowPlaying | null> {
  try {
    const res = await fetch(`${RADIO_API_URL}/api/nowplaying`, {
      cache: 'no-store',
      signal: AbortSignal.timeout(3000),
    })
    if (!res.ok) return null
    const data = await res.json()
    const station = Array.isArray(data) ? data[0] : data
    const song = station?.now_playing?.song
    if (!song) return null
    return {
      title: song.title ?? 'People We Like Radio',
      artist: song.artist ?? '',
    }
  } catch {
    return null
  }
}

export default function RadioPlayer() {
  const audioRef = useRef<HTMLAudioElement>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [nowPlaying, setNowPlaying] = useState<NowPlaying | null>(null)
  const [volume, setVolume] = useState(0.8)

  useEffect(() => {
    fetchNowPlaying().then(setNowPlaying)
    const interval = setInterval(() => fetchNowPlaying().then(setNowPlaying), 30_000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    if (audioRef.current) audioRef.current.volume = volume
  }, [volume])

  async function togglePlay() {
    const audio = audioRef.current
    if (!audio) return
    if (isPlaying) {
      audio.pause()
      audio.src = '' // release stream connection
      setIsPlaying(false)
      setIsLoading(false)
    } else {
      setIsLoading(true)
      audio.src = STREAM_URL
      audio.load()
      try {
        await audio.play()
        setIsPlaying(true)
      } catch {
        // autoplay blocked or stream unavailable — fail silently
      } finally {
        setIsLoading(false)
      }
    }
  }

  return (
    <div className="fixed bottom-0 left-0 right-0 z-50 border-t border-nite-border bg-nite-surface/95 backdrop-blur-sm px-4 py-2 safe-area-inset-bottom">
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <audio ref={audioRef} preload="none" onEnded={() => setIsPlaying(false)} />

      <div className="max-w-lg mx-auto flex items-center gap-3">
        {/* Play / Pause button */}
        <button
          onClick={togglePlay}
          disabled={isLoading}
          aria-label={isPlaying ? 'Pause radio' : 'Play radio'}
          className="shrink-0 w-8 h-8 rounded-full bg-nite-accent text-black flex items-center justify-center
                     hover:bg-amber-400 active:bg-amber-600 disabled:opacity-50 transition-colors"
        >
          {isLoading ? (
            <span className="text-xs font-bold">…</span>
          ) : isPlaying ? (
            <PauseIcon />
          ) : (
            <PlayIcon />
          )}
        </button>

        {/* Station info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-semibold truncate">
              {nowPlaying
                ? nowPlaying.artist
                  ? `${nowPlaying.artist} — ${nowPlaying.title}`
                  : nowPlaying.title
                : 'People We Like Radio'}
            </span>
            {isPlaying && (
              <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />
            )}
          </div>
          <p className="text-xs text-nite-muted">
            {isPlaying ? 'Streaming live' : 'Tap to tune in'}
          </p>
        </div>

        {/* Volume slider */}
        <input
          type="range"
          min="0"
          max="1"
          step="0.05"
          value={volume}
          onChange={(e) => setVolume(Number(e.target.value))}
          aria-label="Volume"
          className="w-16 accent-amber-500 shrink-0"
        />
      </div>
    </div>
  )
}

function PlayIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
      <path d="M3 2.5l11 5.5-11 5.5V2.5z" />
    </svg>
  )
}

function PauseIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
      <rect x="3" y="2" width="3.5" height="12" rx="1" />
      <rect x="9.5" y="2" width="3.5" height="12" rx="1" />
    </svg>
  )
}
