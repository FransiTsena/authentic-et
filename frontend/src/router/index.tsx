import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';

interface RouterContextType {
  path: string;
  navigate: (to: string, options?: { replace?: boolean }) => void;
}

const RouterContext = createContext<RouterContextType>({
  path: window.location.pathname || '/',
  navigate: () => {},
});

export const RouterProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [path, setPath] = useState<string>(() => {
    const p = window.location.pathname || '/';
    return p.endsWith('/') && p.length > 1 ? p.slice(0, -1) : p;
  });

  const navigate = useCallback((to: string, options?: { replace?: boolean }) => {
    const cleanTo = to.endsWith('/') && to.length > 1 ? to.slice(0, -1) : to;
    if (cleanTo === window.location.pathname) return;

    if (options?.replace) {
      window.history.replaceState({}, '', cleanTo);
    } else {
      window.history.pushState({}, '', cleanTo);
    }
    setPath(cleanTo);
    window.scrollTo(0, 0);
  }, []);

  useEffect(() => {
    const handlePopState = () => {
      const current = window.location.pathname || '/';
      const clean = current.endsWith('/') && current.length > 1 ? current.slice(0, -1) : current;
      setPath(clean);
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  return (
    <RouterContext.Provider value={{ path, navigate }}>
      {children}
    </RouterContext.Provider>
  );
};

export const useRouter = () => useContext(RouterContext);

interface LinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  to: string;
  replace?: boolean;
}

export const Link: React.FC<LinkProps> = ({ to, replace, onClick, children, ...rest }) => {
  const { navigate } = useRouter();

  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (onClick) onClick(e);
    // Don't intercept if modifier keys held (Ctrl/Cmd click opens new tab)
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) {
      return;
    }
    e.preventDefault();
    navigate(to, { replace });
  };

  return (
    <a href={to} onClick={handleClick} {...rest}>
      {children}
    </a>
  );
};
