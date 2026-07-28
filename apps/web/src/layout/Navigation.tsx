import { NavLink } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

interface NavItem {
  readonly label: string;
  readonly path: string;
}

const navItems: NavItem[] = [
  { label: "Dashboard", path: "/" },
  { label: "Intents", path: "/intents" },
  { label: "Tasks", path: "/tasks" },
  { label: "Workspaces", path: "/workspaces" },
  { label: "Deployments", path: "/deployments" },
];

const linkBase =
  "rounded-lg px-3 py-2 text-sm font-medium transition-colors duration-150";
const linkActive = "bg-forge-800 text-white";
const linkInactive = "text-slate-400 hover:bg-slate-800 hover:text-slate-100";

export function Navigation(): React.ReactNode {
  const { user, logout } = useAuth();

  return (
    <div className="flex items-center gap-4">
      <nav className="flex items-center gap-1" aria-label="Main navigation">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.path === "/"}
            className={({ isActive }) =>
              `${linkBase} ${isActive ? linkActive : linkInactive}`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>

      {user !== null && (
        <div className="flex items-center gap-3 border-l border-slate-700 pl-4">
          <span className="text-xs text-slate-400">{user.sub}</span>
          <button
            type="button"
            onClick={() => {
              logout().catch(() => undefined);
            }}
            className="rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-100"
          >
            Logout
          </button>
        </div>
      )}
    </div>
  );
}
