/**
 * The approved LocalOps LO monogram (Variant A: the L's foot flows
 * directly into the O's ring, sharing geometry rather than two touching
 * glyphs). Same geometry as the production application icon's mark layer,
 * without its dark-material background — a mark-only treatment for small
 * contexts like the sidebar. Used only by Sidebar, so it's colocated here
 * rather than promoted to shared/ui.
 */
export function LocalOpsMark({ size = 13 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 1024 1024" fill="currentColor" aria-hidden="true">
      <g transform="translate(-109.7,-13.8) scale(1.162)">
        <rect x="260" y="160" width="130" height="480" rx="24" />
        <rect x="260" y="510" width="260" height="130" rx="24" />
        <path
          fillRule="evenodd"
          d="
            M 470,575 A170,170 0 1,0 810,575 A170,170 0 1,0 470,575
            M 592,575 A58,58 0 1,0 708,575 A58,58 0 1,0 592,575"
        />
      </g>
    </svg>
  );
}
