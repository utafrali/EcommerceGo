// ─── Benefit Bar (Modanisa style) ─────────────────────────────────────────────
// 4-column trust/benefit section matching Modanisa's visual style.

const benefits = [
  {
    label: 'Kargo Bedava',
    description: 'Kampanya Koşullarını İncele',
    href: '#',
    icon: (
      <svg width={28} height={28} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M14 18V6a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2v11a1 1 0 0 0 1 1h2" />
        <path d="M15 18H9" />
        <path d="M19 18h2a1 1 0 0 0 1-1v-3.65a1 1 0 0 0-.22-.624l-3.48-4.35A1 1 0 0 0 17.52 8H14" />
        <circle cx={17} cy={18} r={2} />
        <circle cx={7} cy={18} r={2} />
      </svg>
    ),
  },
  {
    label: 'Koşulsuz İade',
    description: 'Masrafsız İade Garantisi',
    href: '#',
    icon: (
      <svg width={28} height={28} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
        <path d="M3 3v5h5" />
        <path d="m9 12 2 2 4-4" />
      </svg>
    ),
  },
  {
    label: 'Kredi Kartı Taksit İmkânı',
    description: 'Detayları İncele',
    href: '#',
    icon: (
      <svg width={28} height={28} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <rect width={20} height={14} x={2} y={5} rx={2} />
        <path d="M2 10h20" />
        <path d="M6 15h2" />
        <path d="M12 15h4" />
      </svg>
    ),
  },
  {
    label: 'Kampanyaları Keşfet',
    description: 'Detayları İncele',
    href: '/products?on_sale=true',
    icon: (
      <svg width={28} height={28} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="m12 2-3.5 7h-5l4 4.5L6 20l6-3 6 3-1.5-6.5 4-4.5h-5z" />
      </svg>
    ),
  },
];

export function BenefitBar() {
  return (
    <section className="border-b border-t border-gray-100 bg-white">
      <div className="mx-auto max-w-screen-xl px-4 sm:px-6">
        <div className="grid grid-cols-2 divide-x divide-y divide-gray-100 md:grid-cols-4 md:divide-y-0">
          {benefits.map((benefit) => (
            <a
              key={benefit.label}
              href={benefit.href}
              className="flex items-center gap-3 px-4 py-4 hover:bg-gray-50 transition-colors sm:px-6"
            >
              <span className="flex-shrink-0 text-brand">{benefit.icon}</span>
              <div>
                <p className="text-[13px] font-semibold text-gray-900">{benefit.label}</p>
                <p className="mt-0.5 text-[11px] text-brand-accent hover:underline">
                  {benefit.description}
                </p>
              </div>
            </a>
          ))}
        </div>
      </div>
    </section>
  );
}
