import { Box, Paper, SxProps, Theme } from '@mui/material';
import Typography from '@mui/material/Typography';

interface Props {
  children: React.ReactNode
  sx?: SxProps<Theme> | undefined
}

function NoResults({ sx, children }: Props) {
  return (
    <Paper variant="outlined" sx={{ display: 'flex', justifyContent: 'center', background: 'inherit', ...sx }}>
      <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
        <Typography color="textSecondary" align="center">
          {children}
        </Typography>
      </Box>
    </Paper>
  );
}

export default NoResults;
