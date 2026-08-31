// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FullscreenCloseButton } from "../src/components/FullscreenCloseButton";

describe("FullscreenCloseButton", () => {
  it("renders an accessible top-level close action and invokes it once", () => {
    const onClose = vi.fn();

    render(<FullscreenCloseButton label="关闭单独训练" onClose={onClose} />);

    const button = screen.getByRole("button", { name: "关闭单独训练" });
    expect(button).toHaveTextContent("×");

    fireEvent.click(button);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not invoke close while disabled", () => {
    const onClose = vi.fn();

    render(<FullscreenCloseButton label="关闭当前页面" onClose={onClose} disabled />);
    fireEvent.click(screen.getByRole("button", { name: "关闭当前页面" }));

    expect(onClose).not.toHaveBeenCalled();
  });
});
