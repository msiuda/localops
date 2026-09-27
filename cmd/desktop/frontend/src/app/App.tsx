import { Sidebar } from "../components/Sidebar";
import { ProjectsPage } from "../features/projects/ProjectsPage";
import "./App.css";

export function App() {
  return (
    <div className="app-shell">
      <Sidebar active="projects" />
      <main className="app-shell__content">
        <ProjectsPage />
      </main>
    </div>
  );
}
