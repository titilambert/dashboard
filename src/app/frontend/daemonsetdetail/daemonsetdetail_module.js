// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import componentsModule from './../common/components/components_module';
import filtersModule from 'common/filters/filters_module';
import logsModule from 'logs/logs_module';
import daemonSetInfo from './daemonsetinfo_directive';
import serviceEndpointDirective from './serviceendpoint_directive';
import stateConfig from './daemonsetdetail_stateconfig';
import sortedHeaderDirective from './sortedheader_directive';
import {DaemonSetService} from './daemonset_service';

/**
 * Angular module for the DaemonSet details view.
 *
 * The view shows detailed view of a DaemonSets.
 */
export default angular
    .module(
        'kubernetesDashboard.daemonSetDetail',
        [
          'ngMaterial',
          'ngResource',
          'ui.router',
          componentsModule.name,
          filtersModule.name,
          logsModule.name,
        ])
    .config(stateConfig)
    .directive('kdDaemonSetInfo', daemonSetInfo)
    .directive('kdServiceEndpoint', serviceEndpointDirective)
    .directive('kdSortedHeader', sortedHeaderDirective)
    .service('kdDaemonSetService', DaemonSetService);
