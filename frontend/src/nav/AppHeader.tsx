import AppBar, { AppBarProps } from '@mui/material/AppBar';
import graphql from 'babel-plugin-relay/macro';
import { Box, Button, Stack } from '@mui/material';
import { styled } from "@mui/material/styles";
import Toolbar from '@mui/material/Toolbar';
import { useEffect, useRef } from 'react';
import AccountMenu from './AccountMenu';
import Link from '../routes/Link';
import { Link as RouterLink } from 'react-router-dom';
import RegistryMenu from './RegistryMenu';
import { useFragment } from 'react-relay/hooks';
import { AppHeaderFragment$key } from './__generated__/AppHeaderFragment.graphql';
import AnnouncementBanner from '../common/AnnouncementBanner';
import { useAppHeaderHeight } from '../contexts/AppHeaderHeightProvider';

const StyledAppBar = styled(AppBar)<AppBarProps>(({ theme }) => ({
    boxShadow: 'none',
    borderBottomStyle: 'solid',
    borderBottomWidth: 1,
    borderBottomColor: theme.palette.divider,
    zIndex: theme.zIndex.drawer + 1,
    backgroundImage: 'none'
}));

interface Props {
    fragmentRef: AppHeaderFragment$key
}

function AppHeader(props: Props) {
    const data = useFragment<AppHeaderFragment$key>(
        graphql`
        fragment AppHeaderFragment on Query
        {
            ...AccountMenuFragment
        }
        `, props.fragmentRef);

    const headerRef = useRef<HTMLDivElement>(null);
    const { setHeaderHeight: setContextHeaderHeight } = useAppHeaderHeight();

    useEffect(() => {
        const updateHeaderHeight = () => {
            if (headerRef.current) {
                const height = headerRef.current.offsetHeight;
                setContextHeaderHeight(height);
            }
        };

        updateHeaderHeight();

        const resizeObserver = new ResizeObserver(updateHeaderHeight);
        if (headerRef.current) {
            resizeObserver.observe(headerRef.current);
        }

        return () => {
            resizeObserver.disconnect();
        };
    }, [setContextHeaderHeight]);

    return (
        <>
            <Box
                ref={headerRef}
                sx={{
                    position: 'fixed',
                    top: 0,
                    left: 0,
                    right: 0,
                    zIndex: 1200,
                }}
            >
                <StyledAppBar position="static" color="inherit">
                    <Toolbar>
                        <Box marginRight={4}>
                            <Link underline="none" color="primary" variant="h5" sx={{ fontWeight: "bold" }} to="/">Tharsis</Link>
                        </Box>
                        <Box display="flex" flex="1" justifyContent="flex-end" alignItems="center">
                            <Stack direction="row" spacing={1} alignItems="center" marginRight={3}>
                                <Button
                                    color="inherit"
                                    sx={{ textTransform: "none", fontWeight: 600 }}
                                    component={RouterLink} to="/groups">
                                    Groups
                                </Button>
                                <Button
                                    color="inherit"
                                    sx={{ textTransform: "none", fontWeight: 600 }}
                                    component={RouterLink} to="/workspaces">
                                    Workspaces
                                </Button>
                                <RegistryMenu />
                            </Stack>
                            <AccountMenu fragmentRef={data} />
                        </Box>
                    </Toolbar>
                </StyledAppBar>
                <AnnouncementBanner />
            </Box>
        </>
    );
}

export default AppHeader;
