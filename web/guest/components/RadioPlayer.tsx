'use client'
// Native radio player — replaces RadioEmbed iframe.
// Streams from NEXT_PUBLIC_RADIO_STREAM_URL (defaults to stream.peoplewelike.club).
// Fetches now-playing metadata from AzuraCast API (non-blocking, non-fatal).
import { useState, useEffect, useRef } from 'react'

const STREAM_URL =
  process.env.NEXT_PUBLIC_RADIO_STREAM_URL ?? 'https://stream.peoplewelike.club/stream'
const RADIO_API_URL =
  process.env.NEXT_PUBLIC_RADIO_API_URL ?? 'https://radio.peoplewelike.club'

interface NowPlaying {
  title: string
  artist: string
  isLive: boolean
}

async function fetchNowPlaying(): Promise<NowPlaying | null> {
  try {
    const res = await fetch(`${RADIO_API_URL}/api/nowplaying`, {
      cache: 'no-store',
      signal: AbortSignal.timeout(3000),
    })
    if (!res.ok) return null
    const data = await res.json()
    // AzuraCast returns an array; first element is the station
    const station = Array.isArray(data) ? data[0] : data
    const song = station?.now_playing?.song
    if (!song) return null
    return {
      title: song.title ?? 'People We Like Radio',
      artist: song.artist ?? '',
      isLive: !!station?.live?.is_live,
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
    // Initial fetch + refresh every 30s
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
    <div className="card">
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <audio ref={audioRef} preload="none" onEnded={() => setIsPlaying(false)} />

      <div className="flex items-center gap-3">
        {/* Play / Pause button */}
        <button
          onClick={togglePlay}
          disabled={isLoading}
          aria-label={isPlaying ? 'Pause radio' : 'Play radio'}
          className="shrink-0 w-10 h-10 rounded-full bg-nite-accent text-black flex items-center justify-center
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
            <span className="text-sm font-semibold">People We Like Radio</span>
            {isPlaying && (
              <span className="inline-flex items-center gap-1 text-xs text-red-400 font-medium">
                <span className="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />
                LIVE
              </span>
            )}
          </div>
          {nowPlaying ? (
            <p className="text-xs text-nite-muted truncate">
              {nowPlaying.artist ? `${nowPlaying.artist} — ` : ''}{nowPlaying.title}
            </p>
          ) : (
            <p className="text-xs text-nite-muted">
              {isPlaying ? 'Streaming…' : 'Tap to tune in'}
            </p>
          )}
        </div>

        {/* Volume slider */}
        <div className="flex items-center gap-1.5 shrink-0">
          <span className="text-xs text-nite-muted">🔈</span>
          <input
            type="range"
            min="0"
            max="1"
            step="0.05"
            value={volume}
            onChange={(e) => setVolume(Number(e.target.value))}
            aria-label="Volume"
            className="w-16 accent-amber-500"
          />
        </div>
      </div>

      {/* External link */}
      <div className="mt-2 flex justify-end">
        <a
          href="https://radio.peoplewelike.club"
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-nite-muted hover:text-nite-text transition-colors"
        >
          Open in browser ↗
        </a>
      </div>
    </div>
  )
}

function PlayIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
      <path d="M3 2.5l11 5.5-11 5.5V2.5z" />
    </svg>
  )
}

function PauseIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
      <rect x="3" y="2" width="3.5" height="12" rx="1" />
      <rect x="9.5" y="2" width="3.5" height="12" rx="1" />
    </svg>
  )
}
