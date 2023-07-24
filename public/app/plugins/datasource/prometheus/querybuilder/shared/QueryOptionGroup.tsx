import { css } from '@emotion/css';
import React from 'react';
import { useToggle } from 'react-use';

import { getValueFormat, GrafanaTheme2 } from '@grafana/data';
import { Stack } from '@grafana/experimental';
import { Collapse, useStyles2 } from '@grafana/ui';
import { QueryStats } from 'app/plugins/datasource/loki/types';

export interface Props {
  title: string;
  collapsedInfo: string[];
  queryStats?: QueryStats | string | null;
  children: React.ReactNode;
}

export function QueryOptionGroup({ title, children, collapsedInfo, queryStats }: Props) {
  const [isOpen, toggleOpen] = useToggle(false);
  const styles = useStyles2(getStyles);

  const generateQueryStats = () => {
    if (typeof queryStats === 'string') {
      return queryStats;
    } else {
      return `This query will process approximately ${convertUnits()}.`;
    }
  };

  const convertUnits = (): string => {
    if (typeof queryStats === 'string' || !queryStats?.bytes) {
      return '';
    }

    const { text, suffix } = getValueFormat('bytes')(queryStats.bytes, 1);
    return text + suffix;
  };

  return (
    <div className={styles.wrapper}>
      <Collapse
        className={styles.collapse}
        collapsible
        isOpen={isOpen}
        onToggle={toggleOpen}
        label={
          <Stack gap={0} wrap={false}>
            <h6 className={styles.title}>{title}</h6>
            {!isOpen && (
              <div className={styles.description}>
                {collapsedInfo.map((x, i) => (
                  <span key={i}>{x}</span>
                ))}
              </div>
            )}
          </Stack>
        }
      >
        <div className={styles.body}>{children}</div>
      </Collapse>
      {queryStats && <p className={styles.stats}>{generateQueryStats()}</p>}
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    collapse: css({
      backgroundColor: 'unset',
      border: 'unset',
      marginBottom: 0,

      ['> button']: {
        padding: theme.spacing(0, 1),
      },
    }),
    wrapper: css({
      width: '100%',
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'baseline',
    }),
    title: css({
      flexGrow: 1,
      overflow: 'hidden',
      fontSize: theme.typography.bodySmall.fontSize,
      fontWeight: theme.typography.fontWeightMedium,
      margin: 0,
    }),
    description: css({
      color: theme.colors.text.secondary,
      fontSize: theme.typography.bodySmall.fontSize,
      fontWeight: theme.typography.bodySmall.fontWeight,
      paddingLeft: theme.spacing(2),
      gap: theme.spacing(2),
      display: 'flex',
    }),
    body: css({
      display: 'flex',
      gap: theme.spacing(2),
      flexWrap: 'wrap',
    }),
    stats: css({
      margin: '0px',
      color: theme.colors.text.secondary,
      fontSize: theme.typography.bodySmall.fontSize,
    }),
  };
};
