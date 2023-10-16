<template>
    <div class="input-with-search">
        <el-row>
            <el-col :span="14">
                <el-input
                        id="input-1"
                        v-model="searchKeyword"
                        placeholder="image digest or attack vector"
                        clearable
                />
            </el-col>

            <el-col :span="5">
                <el-button
                        id="search-1"
                        type="primary"
                        @click="handleSearchImages"
                >
                    Search Image
                </el-button>
            </el-col>

            <el-col :span="4">
                <el-button
                        id="search-2"
                        type="primary"
                        @click="handleSearchVectors"
                >
                    Search Attack Vector
                </el-button>
            </el-col>
        </el-row>
    </div>
    <el-table
            :data="imagesData"
            v-loading="tableLoading1"
            highlight-current-row
            stripe
            table-layout="fixed"
            style="width: 100%"
            max-height="700em"
    >
<!--        可收缩展开内容-->
        <el-table-column fixed type="expand">
            <template #default="props">
                <div>
                  <el-table
                          id="expanded-table"
                          highlight-current-row
                          :data="props.row.layers"
                  >
<!--                      used for left white-->
                      <el-table-column label="" width="40em" />
                      <el-table-column type="index" :index="indexMethod" align="center" label="Index" width="80em" />
                      <el-table-column prop="instruction" label="Instruction" width="450em" />
                      <el-table-column prop="size" label="Size" align="center" width="125em" />
                      <el-table-column prop="digest" label="Digest" width="650em" />
                      <el-table-column prop="results" label="Results" />
                  </el-table>
                </div>
            </template>
        </el-table-column>
        <el-table-column fixed prop="digest" label="Digest" show-overflow-tooltip width="650em" />
        <el-table-column prop="architecture" label="Architecture" show-overflow-tooltip align="center" width="120em" />
        <el-table-column prop="features" label="Features" show-overflow-tooltip align="center" width="100em" />
        <el-table-column prop="variant" label="Variant" show-overflow-tooltip align="center" width="100em" />
        <el-table-column prop="os" label="OS" show-overflow-tooltip align="center" width="100em" />
        <el-table-column prop="size" label="Size" align="center" width="125em" />
        <el-table-column prop="status" label="Status" align="center" width="100em" />
        <el-table-column prop="last_pulled" label="Last Pulled" align="center" width="240em" />
        <el-table-column prop="last_pushed" label="Last Pushed" align="center" width="240em" />
    </el-table>
    <div class="pagination-bottom">
      <el-pagination
              :currentPage="currentPage"
              :page-sizes="[10, 20, 50]"
              :page-size="pageSize"
              layout=" prev, pager, next, jumper, sizes, total, "
              :total="totalCnt"
              @size-change="handleSizeChange"
              @current-change="handleCurrentChange"
              align="center"
      />
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import axios from 'axios';

// create index for each line
const indexMethod = (index: number) => {
    return index + 1
}

const currentPage = ref(1);
const pageSize = ref(20);
const totalCnt = ref(0);    // total count of documents in response
const totalPages = ref(0);  // total count of pages (totalCnt/pageSize + 1)
const searchKeyword = ref('');
const imagesData = ref([]);
const currentSearchMode = ref('handleSearchImages');

// bool value for loading
const tableLoading1 = ref(true);

// get current router
const router = useRouter();

// try to get url query parameter: search
const search = router.currentRoute.value.query.search;
if (search !== undefined) {
    searchKeyword.value = search;
}

function handleSearchImages() {
  // console.log("button clicked");
  // reset to page 1 before every search
  currentPage.value = 1;
  fetchImagesData();
}

function handleSearchVectors() {
    currentPage.value = 1;
    fetchVectorsData();
}

function getImagesData(search, currentPage, pageSize) {
  // axios get images data responsed from backend API
  axios.get('http://10.10.21.212:23434/images', {
      params: {
          search: search,
          page: currentPage,
          page_size: pageSize
      }
  }).then(response => {
      imagesData.value = response.data['results'];
      totalCnt.value = response.data['count'];
      // console.log(imagesData.value);
      // console.log(response.data);
      tableLoading1.value = false;
  })
  .catch(error => {
      console.log(error);
  });
}

function getVectorsData(search, currentPage, pageSize) {
    // axios get images data responsed from backend API
    axios.get('http://10.10.21.212:23434/results', {
        params: {
            search: search,
            page: currentPage,
            page_size: pageSize
        }
    }).then(response => {
        imagesData.value = response.data['results'];
        totalCnt.value = response.data['count'];
        // console.log(imagesData.value);
        // console.log(response.data);
        tableLoading1.value = false;
    })
        .catch(error => {
            console.log(error);
        });
}

function handleCurrentChange(val: number) {
    currentPage.value = val;
    console.log(currentPage.value);
    fetchImagesData();
}

function handleSizeChange(val: number) {
    currentPage.value = 1;
    // change pageSize
    pageSize.value = val;
    console.log(pageSize.value);
    fetchImagesData();
}

// fetch images data from backend with searchKeyword, currentPage and pageSize
// according to image digest
function fetchImagesData() {
    tableLoading1.value = true;
    getImagesData(searchKeyword.value, currentPage.value, pageSize.value);
}

// fetch images data from backend with searchKeyword, currentPage and pageSize
// according to scanning results of images
function fetchVectorsData() {
    tableLoading1.value = true;
    getVectorsData(searchKeyword.value, currentPage.value, pageSize.value);
}

// init web page
fetchImagesData();

</script>

<style scoped>
.input-with-search {
    float: right;
    width: 60%;
}

#search-1 {
    width: 14em;
}

#search-2 {
    width: 14em;
}

.pagination-bottom {
    margin-top: 0.2em;
    display: flex;
    justify-content: center;
}

</style>