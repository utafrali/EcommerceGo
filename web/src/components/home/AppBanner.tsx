// ─── App Download Banner (Modanisa style) ─────────────────────────────────────
// Full-width dark gradient banner promoting mobile app download.

export function AppBanner() {
  return (
    <section className="relative overflow-hidden bg-gradient-to-r from-[#1a1a4e] via-[#16213e] to-[#0f3460]">
      <div className="mx-auto flex max-w-screen-xl items-center justify-between px-8 py-8 sm:px-12 md:px-16">
        {/* Left: Text content */}
        <div className="flex-1">
          <p className="text-sm font-medium text-white/70 sm:text-base">
            Mobil Uygulamaya Özel İlk Siparişine
          </p>
          <h2 className="mt-1 text-2xl font-black text-white sm:text-3xl md:text-4xl">
            EKSTRA %10 İNDİRİM
          </h2>

          {/* Coupon code box */}
          <div className="mt-3 inline-flex items-center gap-2">
            <span className="rounded border border-white/30 bg-white/10 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-white/80">
              KOD
            </span>
            <span className="rounded border-2 border-dashed border-white/50 px-4 py-1 text-xl font-black tracking-widest text-white">
              ILK10
            </span>
          </div>

          <p className="mt-2 text-[11px] text-white/40">
            *1000 TL Ve Üzeri Siparişlerde Geçerlidir.
          </p>

          {/* CTA button */}
          <button
            type="button"
            className="mt-4 rounded bg-white px-6 py-2.5 text-sm font-bold text-[#16213e] hover:bg-white/90 transition-colors"
          >
            UYGULAMAYI İNDİR
          </button>
        </div>

        {/* Right: Decorative phone/visual element */}
        <div className="relative hidden md:flex items-center justify-center">
          {/* Stylized phone outline */}
          <div className="relative flex h-52 w-28 items-start justify-center rounded-[22px] border-4 border-white/20 bg-white/5 pt-4 shadow-2xl">
            {/* Phone notch */}
            <div className="h-4 w-12 rounded-full bg-white/20" />

            {/* App icon grid */}
            <div className="absolute inset-x-3 top-12 grid grid-cols-3 gap-2">
              {['#d63384','#0d6efd','#198754','#f97316','#6f42c1','#dc3545'].map((color, i) => (
                <div
                  key={i}
                  className="h-7 w-7 rounded-xl"
                  style={{ backgroundColor: color }}
                />
              ))}
            </div>

            {/* Bottom indicator */}
            <div className="absolute bottom-3 left-1/2 h-1 w-10 -translate-x-1/2 rounded-full bg-white/30" />
          </div>

          {/* People silhouettes beside phone */}
          <div className="absolute -right-24 bottom-0 flex items-end gap-2">
            <div className="h-40 w-14 rounded-t-full bg-amber-400/30" />
            <div className="h-48 w-14 rounded-t-full bg-orange-300/20" />
          </div>
        </div>
      </div>

      {/* Decorative circles */}
      <div className="absolute -right-8 -top-8 h-32 w-32 rounded-full border border-white/5" />
      <div className="absolute -right-4 -top-4 h-20 w-20 rounded-full border border-white/10" />
      <div className="absolute -bottom-6 left-1/2 h-24 w-24 rounded-full border border-white/5" />
    </section>
  );
}
