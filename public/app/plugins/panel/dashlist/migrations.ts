import { PanelModel } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { FolderDTO } from 'app/types';

import { Options } from './panelcfg.gen';

function getFolderByID(folderID: number) {
  return getBackendSrv().get<FolderDTO>(`/api/folders/id/${folderID}`);
}

export interface AngularModel {
  /** @deprecated */
  starred?: boolean;
  /** @deprecated */
  recent?: boolean;
  /** @deprecated */
  search?: boolean;
  /** @deprecated */
  headings?: boolean;
  /** @deprecated */
  limit?: number;
  /** @deprecated */
  query?: string;
  /** @deprecated */
  folderId?: number;
  /** @deprecated */
  tags?: string[];
}

export async function dashlistMigrationHandler(panel: PanelModel<Options> & AngularModel) {
  const newOptions: Options = {
    ...panel.options,
    showStarred: panel.options.showStarred ?? panel.starred,
    showRecentlyViewed: panel.options.showRecentlyViewed ?? panel.recent,
    showSearch: panel.options.showSearch ?? panel.search,
    showHeadings: panel.options.showHeadings ?? panel.headings,
    maxItems: panel.options.maxItems ?? panel.limit,
    query: panel.options.query ?? panel.query,
    folderId: panel.options.folderId ?? panel.folderId,
    tags: panel.options.tags ?? panel.tags,
  };

  const previousVersion = parseFloat(panel.pluginVersion || '6.1');
  if (previousVersion < 6.3) {
    const oldProps = ['starred', 'recent', 'search', 'headings', 'limit', 'query', 'folderId'] as const;
    oldProps.forEach((prop) => delete panel[prop]);
  }

  if (newOptions.folderId !== undefined) {
    const folderId = newOptions.folderId;
    const folderResp = await getFolderByID(folderId);
    newOptions.folderUID = folderResp.uid;
    delete newOptions.folderId;
  }

  return newOptions;
}
