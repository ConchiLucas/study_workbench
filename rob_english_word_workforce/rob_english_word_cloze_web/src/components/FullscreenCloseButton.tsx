interface FullscreenCloseButtonProps {
  label: string;
  onClose: () => void;
  disabled?: boolean;
}

export function FullscreenCloseButton({ label, onClose, disabled = false }: FullscreenCloseButtonProps) {
  return (
    <button
      className="fullscreen-close-button"
      type="button"
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClose}
    >
      <span aria-hidden="true">×</span>
    </button>
  );
}
