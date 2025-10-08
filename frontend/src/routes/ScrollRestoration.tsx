import { ReactNode, useEffect } from "react";
import { useLocation } from "react-router";

interface Props {
  children: ReactNode
}

const ScrollRestoration = (props: Props) => {
  const location = useLocation();
  useEffect(() => {
    if (!location.state?.preventScrollReset) {
      window.scrollTo(0, 0);
    }
  }, [location]);

  return <>{props.children}</>
};

export default ScrollRestoration;
