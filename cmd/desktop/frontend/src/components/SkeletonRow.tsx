import "./SkeletonRow.css";

export function SkeletonRow() {
  return (
    <div className="skeleton-row" aria-hidden="true">
      <div className="skeleton-row__main">
        <div className="skeleton-block skeleton-row__name" />
        <div className="skeleton-block skeleton-row__path" />
      </div>
      <div className="skeleton-block skeleton-row__side" />
    </div>
  );
}
