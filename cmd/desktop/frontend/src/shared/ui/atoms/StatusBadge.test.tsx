import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { CheckCircle2 } from "lucide-react";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders the label for each tone", () => {
    render(<StatusBadge label="Healthy" tone="success" />);
    expect(screen.getByText("Healthy")).toBeInTheDocument();
  });

  it("renders an icon when provided", () => {
    render(<StatusBadge label="Issues" tone="warning" icon={CheckCircle2} />);
    expect(document.querySelector("svg")).toBeInTheDocument();
  });
});
