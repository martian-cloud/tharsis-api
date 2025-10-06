import DeleteIcon from '@mui/icons-material/Delete';
import { Box, Button, Divider, IconButton, List, ListItem, ListItemText, Typography, useTheme } from "@mui/material";

export type ConfigVersionRunDataOptions = {
    file: any;
};

export const DefaultConfigVersionRunDataOptions: ConfigVersionRunDataOptions = {
    file: null,
};

interface Props {
    data: ConfigVersionRunDataOptions
    onChange: (data: ConfigVersionRunDataOptions) => void
}

function ConfigurationVersionSource({ data, onChange }: Props) {
    const theme = useTheme();

    return (
        <Box mb={4}>
            <Typography variant="subtitle1" gutterBottom>Upload tar.gz file</Typography>
            <Divider light />
            <Box marginTop={2}>
                {data.file ?
                    <List sx={{ border: `1px solid ${theme.palette.divider}`, maxWidth: 400 }}>
                        <ListItem
                            secondaryAction={
                                <IconButton onClick={() => {
                                    onChange({ ...data, file: null })
                                }}>
                                    <DeleteIcon />
                                </IconButton>
                            }>
                            <ListItemText primary={data.file.name} />
                        </ListItem>
                    </List>
                    :
                    <Button
                        variant="outlined"
                        component="label"
                        color="secondary"
                        size="small"
                    >
                        Upload File
                        <input
                            type="file"
                            accept="application/x-gzip"
                            id="tarFile"
                            style={{ display: "none" }}
                            onChange={(event: any) => onChange({ ...data, file: event.target.files[0] })}
                        />
                    </Button>
                }
            </Box>
        </Box>
    );
}

export default ConfigurationVersionSource
