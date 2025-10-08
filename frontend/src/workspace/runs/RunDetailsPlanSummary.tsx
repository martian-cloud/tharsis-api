import MuiInfoIcon from '@mui/icons-material/Info';
import { alpha, Chip, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import { darken, SxProps, Theme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import React, { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { RunDetailsPlanSummaryFragment_plan$key } from './__generated__/RunDetailsPlanSummaryFragment_plan.graphql';
import planDiffColors from './plandiff/RunDetailsPlanDiffColors';

const planSummaryLabels = {
    resourceAdditions: {
        label: (value: number, completed: boolean) => `Resource${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} added`,
        info: 'Number of resources that will be added'
    },
    resourceChanges: {
        label: (value: number, completed: boolean) => `Resource${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} updated`,
        info: 'Number of resources that will be updated'
    },
    resourceDestructions: {
        label: (value: number, completed: boolean) => `Resource${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} destroyed`,
        info: 'Number of resources that will be deleted'
    },
    resourceImports: {
        label: (value: number, completed: boolean) => `Resource${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} imported`,
        info: 'Number of resources that will be imported'
    },
    resourceDrift: {
        label: (value: number) => `Resource${value === 1 ? '' : 's'} drifted`,
        info: 'Number of resources that have drifted'
    },
    outputAdditions: {
        label: (value: number, completed: boolean) => `Output${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} added`,
        info: 'Number of outputs that will be added'
    },
    outputChanges: {
        label: (value: number, completed: boolean) => `Output${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} updated`,
        info: 'Number of outputs that will be updated'
    },
    outputDestructions: {
        label: (value: number, completed: boolean) => `Output${value === 1 ? '' : 's'} ${!completed ? 'will be' : ''} removed`,
        info: 'Number of outputs that will be deleted'
    }
};

const planSummaryColors = {
    resourceAdditions: planDiffColors.create,
    resourceChanges: planDiffColors.update,
    resourceDestructions: planDiffColors.delete,
    resourceImports: planDiffColors.import,
    resourceDrift: planDiffColors.drift,
    outputAdditions: planDiffColors.create,
    outputChanges: planDiffColors.update,
    outputDestructions: planDiffColors.delete
};

function InfoIcon({ text }: { text: string }) {
    return (
        <Tooltip title={text}>
            <MuiInfoIcon sx={{
                width: 16,
                height: 16,
                marginLeft: '10px',
                verticalAlign: 'middle',
                opacity: '20%',
                transition: 'ease',
                transitionDuration: '300ms',
                ":hover": {
                    opacity: '100%'
                }
            }} />
        </Tooltip>
    );
}

function SummaryRowCol({ children, sx }: { children: React.ReactNode, sx?: SxProps<Theme> }) {
    const theme = useTheme();
    return (
        <Paper
            sx={{
                ...sx,
                height: 41,
                pt: 1,
                pb: 1,
                backgroundColor: darken(theme.palette.background.default, 0.3),
                display: 'flex',
                alignItems: 'center',
                width: '100%',
                boxShadow: 'none',
                borderRadius: 0,
                borderBottom: `1px solid ${theme.palette.divider}`
            }}>
            {children}
        </Paper>
    );
}

interface Props {
    ml?: number
    mr?: number
    completed: boolean
    fragmentRef: RunDetailsPlanSummaryFragment_plan$key
}

function RunDetailsPlanStageSummary({ fragmentRef, ml, mr, completed }: Props) {
    const data = useFragment<RunDetailsPlanSummaryFragment_plan$key>(
        graphql`
        fragment RunDetailsPlanSummaryFragment_plan on Plan
        {
            summary {
                resourceAdditions
                resourceChanges
                resourceDestructions
                resourceImports
                resourceDrift
                outputAdditions
                outputChanges
                outputDestructions
            }
        }
      `, fragmentRef);

    const planSummaryItems = useMemo(() => Object.keys(data.summary)
        .filter(key => data.summary[key as keyof typeof data.summary] > 0 && (key !== 'resourceDrift' || !completed))
        .sort()
        .map(key => {
            const value = data.summary[key as keyof typeof data.summary];
            const { label } = planSummaryLabels[key as keyof typeof planSummaryLabels];

            return {
                key,
                value,
                label: label(value, completed),
                info: planSummaryLabels[key as keyof typeof planSummaryLabels].info,
                color: planSummaryColors[key as keyof typeof planSummaryColors]
            };
        }), [data.summary, completed]);

    return (
        <Box ml={ml} mr={mr} display="flex">
            <Box>
                {planSummaryItems.map((item) => (
                    <SummaryRowCol key={item.key} sx={{ pl: 2, pr: 2 }}>
                        <Chip
                            label={`+${item.value}`}
                            size="small"
                            variant="outlined"
                            sx={{ color: item.color, backgroundColor: alpha(item.color, 0.1) }}
                        />
                    </SummaryRowCol>
                ))}
            </Box>
            <Box flex={1}>
                {planSummaryItems.map((item) => (
                    <SummaryRowCol key={item.key} sx={{ pr: 2 }}>
                        <Typography variant="body2" fontWeight={500}>{item.label}</Typography>
                        <Box component="span" flexGrow={1} />
                        <InfoIcon text={item.info} />
                    </SummaryRowCol>
                ))}
            </Box>
        </Box>
    );
}

export default RunDetailsPlanStageSummary;
