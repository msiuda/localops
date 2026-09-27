import { Window } from "@wailsio/runtime";
import "./WindowControls.css";

/**
 * Custom traffic-light-style window controls. LocalOps runs frameless, so
 * there is no native title bar to provide these — this is the app's own
 * close/minimise/maximise affordance, styled to sit in the sidebar.
 */
export function WindowControls() {
  return (
    <div className="window-controls no-drag" role="group" aria-label="Window controls">
      <button
        type="button"
        className="window-controls__dot window-controls__dot--close"
        aria-label="Close window"
        onClick={() => Window.Close()}
      />
      <button
        type="button"
        className="window-controls__dot window-controls__dot--minimise"
        aria-label="Minimise window"
        onClick={() => Window.Minimise()}
      />
      <button
        type="button"
        className="window-controls__dot window-controls__dot--maximise"
        aria-label="Maximise window"
        onClick={() => Window.ToggleMaximise()}
      />
    </div>
  );
}
