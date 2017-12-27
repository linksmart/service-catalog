package eu.linksmart.testing.registration;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.swagger.client.ApiClient;
import io.swagger.client.api.ScApi;
import io.swagger.client.model.APIIndex;
import io.swagger.client.model.Service;
import io.swagger.client.model.ServiceDocs;
import org.junit.Test;

import java.io.File;
import java.math.BigDecimal;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

public class ServiceTesterIT {
    static final String
            BASE_URL = "http://localhost:8082",
            DEFAULT_FILE_NAME = "test/dummy.json";

    static final String
            BASE_URL_PATH = "base_url",
            FILENAME = "filename";
    @Test
    public void registration(){
        if(!System.getenv().containsKey("integration_test"))
            return;
        ApiClient client = new ApiClient();
        ObjectMapper mapper = new ObjectMapper();
        client.setBasePath(System.getenv().getOrDefault(BASE_URL_PATH, BASE_URL));
        ScApi api = new ScApi(client);

        String id = UUID.randomUUID().toString(), file = System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME);


        try{

            Service service = mapper.readValue(new File(file), Service.class);

            Service service1, service2= api.idPut(id,service);

            service1 = api.idGet(id);


            assertTrue("Ids must be equal", id.equals(service1.getId()));
            assertTrue("Name must be equal", service.getName().equals(service1.getName()));
            assertTrue("Description must be equal", service.getDescription().equals(service1.getDescription()));
            assertTrue("Docs must be equal", service.getDocs().equals(service1.getDocs()));

            APIIndex index =  api.rootGet(new BigDecimal(1),new BigDecimal(100));
            assertTrue("It must contain 1 service", index.getTotal().equals(1));

            api.idDelete(id);

        }catch (Exception e){
            e.printStackTrace();
            fail();
        }



    }
}
