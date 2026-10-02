import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { CompactTechStack } from "./CompactTechStack";

describe("CompactTechStack", () => {
  it("renders the capped technologies and a +N overflow chip, never the hidden names directly", () => {
    render(
      <CompactTechStack
        technologies={["TypeScript", "NestJS", "Node.js"]}
        moreTechnologies={["Express", "Jest"]}
        packageManager="yarn"
      />,
    );

    expect(screen.getByText("TypeScript")).toBeInTheDocument();
    expect(screen.getByText("NestJS")).toBeInTheDocument();
    expect(screen.getByText("Node.js")).toBeInTheDocument();
    expect(screen.getByText("+2")).toBeInTheDocument();
    expect(screen.queryByText("Express")).not.toBeInTheDocument();
    expect(screen.queryByText("Jest")).not.toBeInTheDocument();
  });

  it("exposes the hidden technology names through the overflow chip's title", () => {
    render(<CompactTechStack technologies={["Go"]} moreTechnologies={["Gin", "Echo"]} />);

    expect(screen.getByTitle("Gin, Echo")).toBeInTheDocument();
  });

  it("renders no overflow chip when nothing is hidden", () => {
    render(<CompactTechStack technologies={["Go"]} moreTechnologies={[]} />);

    expect(screen.queryByText(/^\+/)).not.toBeInTheDocument();
  });

  it("renders the package manager as subordinate text alongside the technology chips", () => {
    render(<CompactTechStack technologies={["PHP", "Laravel"]} packageManager="composer" />);

    expect(screen.getByText("PHP")).toBeInTheDocument();
    expect(screen.getByText("Laravel")).toBeInTheDocument();
    expect(screen.getByText(/composer/)).toBeInTheDocument();
  });

  it("renders nothing when there are no technologies and no package manager", () => {
    const { container } = render(<CompactTechStack technologies={[]} moreTechnologies={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
