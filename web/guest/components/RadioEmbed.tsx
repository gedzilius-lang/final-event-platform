// Radio embed — shows the People We Like radio player.
// The radio runs on a separate VPS (radio.peoplewelike.club) — we embed it, never host it.
// Per PRODUCT_BLUEPRINT: "embedded in Global Mode guest app as an iframe/embed".
export default function RadioEmbed() {
  return (
    <div className="card">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="text-base">📻</span>
          <span className="text-sm font-semibold">People We Like Radio</span>
        </div>
        <a
          href="https://radio.peoplewelike.club"
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-nite-muted hover:text-nite-text transition-colors"
        >
          Open ↗
        </a>
      </div>
      <iframe
        src="https://radio.peoplewelike.club/embed"
        width="100%"
        height="80"
        frameBorder="0"
        scrolling="no"
        allow="autoplay"
        className="rounded-lg bg-nite-bg"
        title="People We Like Radio"
      />
    </div>
  )
}
