import { lighten, styled } from '@mui/material';

export const StyledCode = styled(
    'code'
)(({ theme }) => ({
    padding: "2px 4px",
    color: `${theme.palette.text.primary}`,
    backgroundColor: lighten(theme.palette.background.paper, 0.2),
    borderRadius: "4px",
    fontSize: "90%",
    whiteSpace: "nowrap",
}));
