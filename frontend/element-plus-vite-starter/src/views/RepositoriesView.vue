<template>
    <div class="input-with-search">
        <el-input id="input-1" v-model="searchKeyword" placeholder="namespace, name or description of repository" clearable>
            <template #append>
                <el-button id="search-1" type="primary" @click="handleSearchRepositories">Search</el-button>
            </template>
        </el-input>
    </div>
    <el-table
        :data="repositoriesData"
        highlight-current-row
        stripe
        table-layout="fixed"
        style="width: 100%"
        max-height="700"
    >
        <!--        可收缩展开内容-->
        <el-table-column fixed type="expand">
            <template #default="repoProps">
                <div>
                    <el-table
                        id="expanded-tags-table"
                        highlight-current-row
                        :data="repoProps.row.tags"
                        :row-class-name="tableRowClassName"
                    >
                        <el-table-column fixed label="" width="50" />
                        <el-table-column fixed type="expand">
                            <template #default="tagProps">
                                <div>
                                    <el-table
                                        id="expanded-images-table"
                                        highlight-current-row
                                        :data="tagProps.row.images"
                                        :row-class-name="tableRowClassName"
                                    >
                                        <el-table-column label="" width="150" />
                                        <el-table-column prop="colId" label="Index" align="center" width="200" />
                                        <el-table-column prop="architecture" label="Architecture" align="center" width="200" />
                                        <el-table-column prop="variant" label="Variant" align="center" width="400" />
                                        <el-table-column prop="digest" label="Digest" width="1000" />
                                    </el-table>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column prop="colId" label="Index" align="center" width="80" />
                        <el-table-column prop="tag_name" label="Tag Name" align="center" width="200" />
                        <el-table-column prop="tag_last_pulled" label="Last Updated" align="center" width="240" />
                        <el-table-column prop="last_updater_username" label="Last Updater" align="center" show-overflow-tooltip width="200" />
                        <el-table-column prop="tag_last_pulled" label="Last Pulled" align="center" width="240" />
                        <el-table-column prop="tag_last_pushed" label="Last Pushed" align="center" width="240" />
                        <el-table-column prop="media_type" label="Media Type" show-overflow-tooltip align="center" width="400" />
                        <el-table-column prop="content_type" label="Content Type" align="center" width="150" />
                    </el-table>
                </div>
            </template>
        </el-table-column>
        <el-table-column fixed prop="namespace" label="Namespace" align="center" show-overflow-tooltip width="200" />
        <el-table-column fixed prop="name" label="Name" align="center" show-overflow-tooltip width="200" />
        <el-table-column prop="user" label="User" align="center" show-overflow-tooltip width="200" />
        <el-table-column prop="repository_type" label="Repo Type" align="center" width="100" />
        <el-table-column prop="description" label="Description" show-overflow-tooltip width="300" />
        <el-table-column prop="star_count" label="Star Count" align="center" width="100" />
        <el-table-column prop="pull_count" label="Pull Count" align="center" width="125" />
        <el-table-column prop="date_registered" label="Date Registered" align="center" width="240" />
        <el-table-column prop="last_updated" label="Last Updated" align="center" width="240" />
        <el-table-column prop="full_description" label="Full Description" show-overflow-tooltip width="600" />
    </el-table>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import axios from 'axios';

const tableRowClassName = ({row, rowIndex}) => {
    row.colId = rowIndex + 1;
};

const page = ref(1);
const pageSize = ref(20);
const searchKeyword = ref('');
const repositoriesData = ref([]);

function handleSearchRepositories() {
    console.log('search repositories');
    // reset to page 1 before every search
    page.value = 1;
    getRepositoriesData(searchKeyword.value, page.value, pageSize.value);
}

function getRepositoriesData(search, page, pageSize) {
    // axios get images data responsed from backend API
    axios.get('http://10.10.21.122:23434/repositories', {
        params: {
            search: search,
            page: page,
            page_size: pageSize
        }
    }).then(response => {
        repositoriesData.value = response.data['results'];
        // console.log(imagesData.value);
        // console.log(response.data);
    })
    .catch(error => {
        console.log(error);
    });
}

getRepositoriesData(searchKeyword.value, page.value, pageSize.value);
</script>

<style scoped>
.input-with-search {
    float: right;
    width: 45%;
}
</style>