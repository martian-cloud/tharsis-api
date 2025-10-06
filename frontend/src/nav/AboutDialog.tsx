import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, List, ListItem, ListItemText, Typography } from "@mui/material";
import Timestamp from "../common/Timestamp";

interface Props {
    version: string;
    buildTimestamp: string;
    dbMigrationVersion: string;
    dbMigrationDirty: boolean;
    onClose: () => void;
}

function AboutDialog({
    version,
    buildTimestamp,
    dbMigrationVersion,
    dbMigrationDirty,
    onClose
}: Props) {
    const stripPrefix = (version: string) => {
        const prefix = 'v'
        return version.startsWith(prefix) ? version.slice(prefix.length) : version;
    }

    return (
        <Dialog
            fullWidth
            maxWidth="xs"
            open={true}
        >
            <DialogTitle sx={{ m: 0, p: 2 }}>
                About Tharsis
            </DialogTitle>
            <DialogContent dividers>
                <Typography gutterBottom variant="subtitle2">
                    An enterprise scale Terraform platform that offers a complete solution for managing your Terraform deployments, state and workspaces.
                </Typography>
                <Box>
                    <List sx={{ padding: 0 }}>
                        <ListItem sx={{ padding: 0 }}>
                            <ListItemText
                                primary="Version"
                                secondary={
                                    <>
                                        {stripPrefix(version)} &mdash; <Timestamp
                                            tooltip="Backend and frontend build date"
                                            format="absolute"
                                            timestamp={buildTimestamp}
                                        />
                                    </>
                                }
                            />
                        </ListItem>
                        <ListItem sx={{ padding: 0 }}>
                            <ListItemText
                                primary="Database Migration"
                                secondary={
                                    <>
                                        {dbMigrationVersion}
                                        {dbMigrationDirty && <strong> (dirty)</strong>}
                                    </>
                                }
                            />
                        </ListItem>
                    </List>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button autoFocus onClick={onClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
}

export default AboutDialog;
