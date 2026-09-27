import "./TechPill.css";

interface TechPillProps {
  label: string;
}

export function TechPill({ label }: TechPillProps) {
  return <span className="tech-pill">{label}</span>;
}
