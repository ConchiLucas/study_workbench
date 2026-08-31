import { FullscreenCloseButton } from "./FullscreenCloseButton";

interface ReviewPracticeLauncherProps {
  dueCount: number;
  wrongCount: number;
  loading: boolean;
  onStart: () => void;
  onOpenWrongSentences: () => void;
  onOpenSolo: () => void;
}

export function ReviewPracticeLauncher({
  dueCount,
  wrongCount,
  loading,
  onStart,
  onOpenWrongSentences,
  onOpenSolo,
}: ReviewPracticeLauncherProps) {
  return (
    <section className="practice-launcher review-practice-launcher">
      <div className="practice-launcher-copy">
        <span className="launcher-kicker">REVIEW</span>
        <h2>错题挖空复习</h2>
        <p>{dueCount > 0 ? `今天有 ${dueCount} 道到期错题需要复习` : "今天没有到期错题"}</p>
      </div>
      <div className="launcher-controls review-launcher-controls">
        <button className="primary-button start-practice-btn" type="button" onClick={onStart} disabled={loading || dueCount === 0}>
          开始答题
        </button>
        <div className="review-secondary-actions">
          <button className="ghost-button wrong-collection-btn" type="button" onClick={onOpenWrongSentences} disabled={loading}>
            错题集 {wrongCount}
          </button>
          <button className="ghost-button solo-entry-btn" type="button" onClick={onOpenSolo} disabled={loading}>
            单独训练
          </button>
        </div>
      </div>
    </section>
  );
}

interface SoloTrainingLauncherProps {
  selectedLabel: string;
  batchText: string;
  loading: boolean;
  showClose: boolean;
  onClose: () => void;
  onChooseDifficulty: () => void;
  onOpenSentences: () => void;
  onOpenResults: () => void;
  onStart: () => void;
}

export function SoloTrainingLauncher(props: SoloTrainingLauncherProps) {
  return (
    <section className="practice-launcher solo-training-launcher">
      {props.showClose ? <FullscreenCloseButton label="关闭单独训练" onClose={props.onClose} /> : null}
      <div className="practice-launcher-copy">
        <span className="launcher-kicker">SOLO TRAINING</span>
        <h2>单独训练</h2>
        <p>选择词库难度，进行独立的句子挖空训练</p>
      </div>
      <div className="launcher-controls">
        <div className="selected-difficulty-pill"><span>{props.selectedLabel}</span><small>{props.batchText}</small></div>
        <button className="ghost-button" type="button" onClick={props.onChooseDifficulty}>选择难度</button>
        <button className="ghost-button" type="button" onClick={props.onOpenSentences}>句子列表</button>
        <button className="ghost-button" type="button" onClick={props.onOpenResults}>答题结果</button>
        <button className="primary-button" type="button" onClick={props.onStart} disabled={props.loading}>开始训练</button>
      </div>
    </section>
  );
}
